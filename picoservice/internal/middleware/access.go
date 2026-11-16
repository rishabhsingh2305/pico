package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/authz"
	"picoservice/internal/entities"
	"picoservice/internal/httpx"
)

// RequireOrgAccess authorizes an organization-scoped request. It requires the
// orgId query parameter, at least one feature role for that organization in the
// access token, and the unit permission declared for the matched route. It
// reads no database state; role changes take effect when a token is reissued.
//
// RequireAuth must run first.
func RequireOrgAccess(logger *slog.Logger, policy *authz.Policy) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := FromContext(r.Context())
			if !ok || id.UserID == "" {
				httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
				return
			}

			orgID, ok := queryID(w, r, OrgIDParam)
			if !ok {
				return
			}
			if pathOrgID := chi.URLParam(r, "orgID"); pathOrgID != "" && pathOrgID != orgID {
				httpx.WriteError(w, r, http.StatusBadRequest, "orgId must match the path organization")
				return
			}

			roles, ok := id.Orgs[orgID]
			if !ok || len(roles) == 0 {
				httpx.WriteError(w, r, http.StatusForbidden, "forbidden")
				return
			}

			pattern := chi.RouteContext(r.Context()).RoutePattern()
			endpoint, ok := policy.Endpoint(r.Method, pattern)
			if !ok || endpoint.Scope != authz.ScopeOrg {
				logger.ErrorContext(r.Context(), "organization route has no authz policy",
					"method", r.Method,
					"pattern", pattern,
					"request_id", httpx.RequestID(r.Context()),
				)
				httpx.WriteError(w, r, http.StatusInternalServerError, "internal server error")
				return
			}

			if !allowsAnyRole(policy, roles, endpoint.Permission) {
				httpx.WriteError(w, r, http.StatusForbidden, "forbidden")
				return
			}

			id.OrgID = orgID
			next.ServeHTTP(w, r.WithContext(WithContext(r.Context(), id)))
		})
	}
}

func allowsAnyRole(policy *authz.Policy, roles []string, permission string) bool {
	for _, role := range roles {
		if role == entities.RoleWildcard {
			return true
		}
		if policy.Allow(role, permission) {
			return true
		}
	}
	return false
}
