package routestest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"picoservice/internal/authz"
	"picoservice/internal/entities"
	"picoservice/test/fakes"
)

// TestAuthorizationMatrix exercises every org-scope route with every declared
// feature role. A role either grants the route permission or receives 403; a
// missing or misconfigured policy would surface as 500.
func TestAuthorizationMatrix(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	replacements := map[string]string{
		"{orgID}":        "org-1",
		"{projectID}":    "project-1",
		"{expenseID}":    "expense-1",
		"{memberUserID}": "user-2",
	}
	roles := append(policy.Roles(), entities.RoleWildcard)

	for _, endpoint := range policy.Endpoints() {
		if endpoint.Scope != authz.ScopeOrg {
			continue
		}

		path := endpoint.Path
		for placeholder, value := range replacements {
			path = strings.ReplaceAll(path, placeholder, value)
		}
		path += "?orgId=org-1"
		if endpoint.Permission == "shares.status" {
			path += "&resourceType=expense&resourceIds=expense-1"
		}

		for _, role := range roles {
			role := role
			t.Run(endpoint.Permission+"/"+role, func(t *testing.T) {
				fakeServices := fakes.NewServices()
				fakeServices.Auth.Orgs = map[string]map[string][]string{
					"user-1": {"org-1": {role}},
				}
				r := newRouter(t, fakeServices, stubPinger{})

				req := httptest.NewRequest(endpoint.Method, path, strings.NewReader(`{}`))
				req.Header.Set("Authorization", "Bearer user-1")
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)

				allowed := role == entities.RoleWildcard || policy.Allow(role, endpoint.Permission)
				if allowed {
					if rec.Code >= http.StatusBadRequest {
						t.Fatalf("expected %s to allow %s, got %d", role, endpoint.Permission, rec.Code)
					}
					return
				}
				if rec.Code != http.StatusForbidden {
					t.Fatalf("expected %s to deny %s with 403, got %d", role, endpoint.Permission, rec.Code)
				}
			})
		}
	}
}
