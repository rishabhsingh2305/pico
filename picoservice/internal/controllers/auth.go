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
	Refresh(ctx context.Context, token string) (services.LoginResult, error)
	OrgRoles(token string) (map[string][]string, error)
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

	c.writeAuthResponse(w, r, result)
}

// Refresh handles POST /auth/refresh. It accepts the refresh token in the JSON
// body or the refresh cookie and rotates both tokens.
func (c *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	token := refreshToken(w, r)
	if token == "" {
		httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
		return
	}

	result, err := c.service.Refresh(r.Context(), token)
	if err != nil {
		if errors.Is(err, entities.ErrInvalidToken) {
			middleware.ClearAuthCookie(w)
			middleware.ClearRefreshCookie(w)
		}
		c.writeError(w, r, err)
		return
	}

	c.writeAuthResponse(w, r, result)
}

func refreshToken(w http.ResponseWriter, r *http.Request) string {
	var request refreshRequest
	if err := httpx.DecodeJSON(w, r, &request); err == nil && request.RefreshToken != "" {
		return request.RefreshToken
	}
	if cookie, err := r.Cookie(middleware.RefreshCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

func (c *AuthController) writeAuthResponse(w http.ResponseWriter, r *http.Request, result services.LoginResult) {
	c.setAuthCookie(w, result.Token, result.ExpiresAt)
	c.setRefreshCookie(w, result.RefreshToken, result.RefreshExpiresAt)
	httpx.WriteJSON(w, r, http.StatusOK, authResponse{
		AccessToken:      result.Token,
		TokenType:        "Bearer",
		User:             userResponseFrom(result.User),
		ExpiresAt:        result.ExpiresAt,
		RefreshToken:     result.RefreshToken,
		RefreshExpiresAt: &result.RefreshExpiresAt,
		Orgs:             result.Orgs,
	})
}

// Logout handles POST /auth/logout. Clearing the cookies is the only
// server-side action because access tokens are stateless.
func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearAuthCookie(w)
	middleware.ClearRefreshCookie(w)
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

	orgs, _ := c.service.OrgRoles(token)
	httpx.WriteJSON(w, r, http.StatusOK, authResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        userResponseFrom(user),
		ExpiresAt:   expiresAt,
		Orgs:        orgs,
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
	c.setCookie(w, middleware.AuthCookieName, token, expiresAt)
}

func (c *AuthController) setRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	c.setCookie(w, middleware.RefreshCookieName, token, expiresAt)
}

func (c *AuthController) setCookie(w http.ResponseWriter, name, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
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

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authResponse struct {
	AccessToken      string              `json:"accessToken"`
	TokenType        string              `json:"tokenType"`
	User             userResponse        `json:"user"`
	ExpiresAt        time.Time           `json:"expiresAt"`
	RefreshToken     string              `json:"refreshToken,omitempty"`
	RefreshExpiresAt *time.Time          `json:"refreshExpiresAt,omitempty"`
	Orgs             map[string][]string `json:"orgs,omitempty"`
}
