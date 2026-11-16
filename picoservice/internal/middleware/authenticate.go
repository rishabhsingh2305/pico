package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"picoservice/internal/httpx"
)

// Authenticate accepts either a user JWT or a share credential and stores the
// resulting identity in the request context.
func Authenticate(logger *slog.Logger, tokens AccessTokens, credentials CredentialResolver) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := shareToken(r); token != "" {
				credential, err := credentials.ResolveToken(r.Context(), token)
				if err != nil {
					httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
					return
				}
				scope := credential.ShareScope()
				ctx := WithContext(r.Context(), Identity{
					UserID:  "share:" + credential.ID,
					OrgID:   credential.OrgID,
					Kind:    IdentityKindShare,
					ShareID: credential.ID,
					Scopes:  scopeSet(credential.Scopes),
					Share:   &scope,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

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

func shareToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Share") {
		return ""
	}
	return strings.TrimSpace(token)
}

func scopeSet(scopes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		set[scope] = struct{}{}
	}
	return set
}
