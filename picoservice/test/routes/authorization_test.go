package routestest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/authz"
	"picoservice/internal/entities"
	"picoservice/test/fakes"
)

func TestRoutePolicyCoverage(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	r := newTestRouter(t)
	routed := make(map[string]struct{})
	err = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routed[walkKey(method, route)] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	for key := range routed {
		method, route, _ := strings.Cut(key, " ")
		if _, ok := policy.Endpoint(method, route); !ok {
			t.Errorf("route %s %s has no authz policy", method, route)
		}
	}

	for _, endpoint := range policy.Endpoints() {
		if _, ok := routed[walkKey(endpoint.Method, endpoint.Path)]; !ok {
			t.Errorf("policy %s %s does not match a route", endpoint.Method, endpoint.Path)
		}
	}
}

func TestProtectedRoutesRequireAuthentication(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	r := newTestRouter(t)
	replacements := map[string]string{
		"{orgID}":        "org-1",
		"{projectID}":    "project-1",
		"{expenseID}":    "expense-1",
		"{memberUserID}": "user-2",
		"{linkID}":       "link-1",
	}

	for _, endpoint := range policy.Endpoints() {
		if endpoint.Scope == authz.ScopePublic {
			continue
		}

		path := endpoint.Path
		for placeholder, value := range replacements {
			path = strings.ReplaceAll(path, placeholder, value)
		}
		if endpoint.Scope == authz.ScopeOrg {
			path += "?orgId=org-1"
		}

		t.Run(endpoint.Permission, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(endpoint.Method, path, nil)
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected %s %s to require authentication, got %d", endpoint.Method, path, rec.Code)
			}
		})
	}
}

func TestRequireAuthOnRoutes(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		invalid       bool
		path          string
		wantStatus    int
	}{
		{name: "missing", path: "/users/me", wantStatus: http.StatusUnauthorized},
		{name: "invalid", authorization: "Bearer bad-token", invalid: true, path: "/users/me", wantStatus: http.StatusUnauthorized},
		{name: "valid", authorization: "Bearer user-1", path: "/users/me", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServices := fakes.NewServices()
			if tt.invalid {
				fakeServices.Auth.ParseAccessFunc = func(string) (string, map[string][]string, error) {
					return "", nil, entities.ErrInvalidToken
				}
			}
			fakeServices.Users.GetFunc = func(context.Context, string) (entities.User, error) {
				return entities.User{ID: "user-1"}, nil
			}
			r := newRouter(t, fakeServices, stubPinger{})
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestRequireOrgAccess(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		orgs       map[string][]string
		body       string
		wantStatus int
	}{
		{
			name:       "missing identity",
			method:     http.MethodGet,
			path:       "/projects?orgId=org-1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing orgId",
			method:     http.MethodGet,
			path:       "/projects",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {entities.RoleWildcard}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no role for org",
			method:     http.MethodGet,
			path:       "/projects?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-2": {entities.RoleWildcard}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "viewer reads",
			method:     http.MethodGet,
			path:       "/projects?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"project.viewer"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "viewer cannot write",
			method:     http.MethodPost,
			path:       "/projects?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"project.viewer"}},
			body:       `{"name":"Launch","currency":"USD"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "editor writes",
			method:     http.MethodPost,
			path:       "/projects?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"project.editor"}},
			body:       `{"name":"Launch","currency":"USD"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "mixed roles cannot write projects",
			method:     http.MethodPost,
			path:       "/projects?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"project.viewer", "expense.editor"}},
			body:       `{"name":"Launch","currency":"USD"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "mixed roles write expenses",
			method:     http.MethodPost,
			path:       "/expenses?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"project.viewer", "expense.editor"}},
			body:       `{"projectId":"project-1","amount":100,"spentAt":"2026-01-02T03:04:05Z"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "editor cannot manage members",
			method:     http.MethodPost,
			path:       "/orgs/org-1/members?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"member.viewer"}},
			body:       `{"userId":"user-2"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "member admin adds members",
			method:     http.MethodPost,
			path:       "/orgs/org-1/members?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"member.admin"}},
			body:       `{"userId":"user-2"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "member admin changes roles",
			method:     http.MethodPatch,
			path:       "/orgs/org-1/members/user-2?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"member.admin"}},
			body:       `{"roles":{"project":"viewer"}}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "org admin updates the organization",
			method:     http.MethodPatch,
			path:       "/orgs/org-1?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"org.admin"}},
			body:       `{"name":"Acme Labs"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "org viewer cannot update the organization",
			method:     http.MethodPatch,
			path:       "/orgs/org-1?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {"org.viewer"}},
			body:       `{"name":"Acme Labs"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "owner writes",
			method:     http.MethodPost,
			path:       "/projects?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {entities.RoleWildcard}},
			body:       `{"name":"Launch","currency":"USD"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "org path and query mismatch",
			method:     http.MethodGet,
			path:       "/orgs/org-1?orgId=org-2",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {entities.RoleWildcard}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "org path and query match",
			method:     http.MethodGet,
			path:       "/orgs/org-1?orgId=org-1",
			token:      "user-1",
			orgs:       map[string][]string{"org-1": {entities.RoleWildcard}},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServices := fakes.NewServices()
			fakeServices.Auth.Orgs = map[string]map[string][]string{}
			if tt.token != "" {
				fakeServices.Auth.Orgs[tt.token] = tt.orgs
			}
			r := newRouter(t, fakeServices, stubPinger{})

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
