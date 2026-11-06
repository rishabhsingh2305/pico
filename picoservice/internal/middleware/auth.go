package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"picoservice/internal/httpx"
)

// AuthCookieName is the cookie carrying the access token.
const AuthCookieName = "pico_auth"

// AccessTokens validates an access token and returns its subject.
type AccessTokens interface {
	ParseSubject(token string) (string, error)
}

// RequireAuth authenticates the request from the auth cookie. Requests without a
// cookie fall back to the temporary userId query parameter until the UI
// migrates to tokens. A present but invalid cookie never falls back.
func RequireAuth(logger *slog.Logger, users UserLookup, tokens AccessTokens) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := bearerToken(r); token != "" {
				userID, err := tokens.ParseSubject(token)
				if err != nil {
					httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
					return
				}

				ctx := WithContext(r.Context(), Identity{UserID: userID})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if cookie, err := r.Cookie(AuthCookieName); err == nil && cookie.Value != "" {
				userID, err := tokens.ParseSubject(cookie.Value)
				if err != nil {
					ClearAuthCookie(w)
					httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
					return
				}

				ctx := WithContext(r.Context(), Identity{UserID: userID})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			userID := strings.TrimSpace(r.URL.Query().Get(UserIDParam))
			if userID == "" {
				httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
				return
			}
			if len(userID) > maxIDLength {
				httpx.WriteError(w, r, http.StatusBadRequest, UserIDParam+" must be at most 128 characters")
				return
			}

			lookupCtx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
			defer cancel()

			exists, err := users.Exists(lookupCtx, userID)
			if err != nil {
				writeLookupError(logger, w, r, "user lookup failed", err)
				return
			}
			if !exists {
				httpx.WriteError(w, r, http.StatusNotFound, "user not found")
				return
			}

			ctx := WithContext(r.Context(), Identity{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// ClearAuthCookie expires the auth cookie.
func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
