package middleware

import (
	"net/http"
	"strings"
	"time"

	"picoservice/internal/httpx"
)

// AuthCookieName is the cookie carrying the access token.
const AuthCookieName = "pico_auth"

// RefreshCookieName is the cookie carrying the refresh token.
const RefreshCookieName = "pico_refresh"

// AccessTokens validates an access token and returns its subject and
// organization role map.
type AccessTokens interface {
	ParseAccess(token string) (subject string, orgs map[string][]string, err error)
}

// RequireAuth authenticates the request from a Bearer token or the auth cookie
// and stores the identity in the request context. There is no query-parameter
// fallback: the actor always comes from a signed token.
func RequireAuth(tokens AccessTokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := bearerToken(r); token != "" {
				authenticate(w, r, next, tokens, token, false)
				return
			}

			if cookie, err := r.Cookie(AuthCookieName); err == nil && cookie.Value != "" {
				authenticate(w, r, next, tokens, cookie.Value, true)
				return
			}

			httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
		})
	}
}

func authenticate(w http.ResponseWriter, r *http.Request, next http.Handler, tokens AccessTokens, token string, clearCookie bool) {
	subject, orgs, err := tokens.ParseAccess(token)
	if err != nil {
		if clearCookie {
			ClearAuthCookie(w)
		}
		httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
		return
	}

	ctx := WithContext(r.Context(), Identity{UserID: subject, Orgs: orgs})
	next.ServeHTTP(w, r.WithContext(ctx))
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
	clearCookie(w, AuthCookieName)
}

// ClearRefreshCookie expires the refresh cookie.
func ClearRefreshCookie(w http.ResponseWriter) {
	clearCookie(w, RefreshCookieName)
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
