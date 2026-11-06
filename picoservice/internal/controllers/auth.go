package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/middleware"
	"picoservice/internal/services"
)

// AuthService is the authentication behavior used by the auth controller.
type AuthService interface {
	Login(ctx context.Context, email, password string) (services.LoginResult, error)
	Session(ctx context.Context, token string) (entities.User, time.Time, error)
}

// AuthController serves login, logout, and session endpoints.
type AuthController struct {
	logger        *slog.Logger
	service       AuthService
	secureCookies bool
}

// NewAuthController returns an auth controller backed by service. secureCookies
// controls the Secure attribute on the auth cookie.
func NewAuthController(logger *slog.Logger, service AuthService, secureCookies bool) *AuthController {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthController{logger: logger, service: service, secureCookies: secureCookies}
}

// Login handles POST /auth/login.
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := c.service.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	c.setAuthCookie(w, result.Token, result.ExpiresAt)
	httpx.WriteJSON(w, r, http.StatusOK, authResponse{
		AccessToken: result.Token,
		TokenType:   "Bearer",
		User:        userResponseFrom(result.User),
		ExpiresAt:   result.ExpiresAt,
	})
}

// Logout handles POST /auth/logout. Clearing the cookie is the only server-side
// action because access tokens are stateless.
func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearAuthCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Session handles GET and HEAD /auth/session.
func (c *AuthController) Session(w http.ResponseWriter, r *http.Request) {
	token := accessToken(r)
	if token == "" {
		httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
		return
	}

	user, expiresAt, err := c.service.Session(r.Context(), token)
	if err != nil {
		if errors.Is(err, entities.ErrInvalidToken) {
			middleware.ClearAuthCookie(w)
		}
		c.writeError(w, r, err)
		return
	}

	httpx.WriteJSON(w, r, http.StatusOK, authResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        userResponseFrom(user),
		ExpiresAt:   expiresAt,
	})
}

func accessToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if scheme, token, ok := strings.Cut(header, " "); ok && strings.EqualFold(scheme, "Bearer") {
		if token = strings.TrimSpace(token); token != "" {
			return token
		}
	}
	if cookie, err := r.Cookie(middleware.AuthCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

func (c *AuthController) setAuthCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secureCookies,
		SameSite: c.sameSite(),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (c *AuthController) sameSite() http.SameSite {
	if c.secureCookies {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func (c *AuthController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, entities.ErrInvalidCredentials):
		httpx.WriteError(w, r, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, entities.ErrEmailNotVerified):
		httpx.WriteError(w, r, http.StatusForbidden, "email not verified")
	case errors.Is(err, entities.ErrInvalidToken):
		httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
	default:
		logInternalError(c.logger, w, r, "auth request failed", err)
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	User        userResponse `json:"user"`
	ExpiresAt   time.Time    `json:"expiresAt"`
}
