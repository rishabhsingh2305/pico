package middlewaretest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/authz"
	"picoservice/internal/entities"
	"picoservice/internal/middleware"
)

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		cookie        string
		authorization string
		tokens        fakeAccessTokens
		wantStatus    int
		wantUser      string
	}{
		{
			name:       "missing identity",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "query userId is ignored",
			query:      "?userId=user-1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "valid bearer",
			authorization: "Bearer token-1",
			tokens:        fakeAccessTokens{subject: "user-1"},
			wantStatus:    http.StatusOK,
			wantUser:      "user-1",
		},
		{
			name:          "invalid bearer does not fall back",
			query:         "?userId=user-1",
			authorization: "Bearer bad-token",
			tokens:        fakeAccessTokens{err: errors.New("invalid token")},
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:       "valid cookie",
			cookie:     "token-1",
			tokens:     fakeAccessTokens{subject: "user-1"},
			wantStatus: http.StatusOK,
			wantUser:   "user-1",
		},
		{
			name:       "invalid cookie does not fall back",
			query:      "?userId=user-1",
			cookie:     "bad-token",
			tokens:     fakeAccessTokens{err: errors.New("invalid token")},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var got middleware.Identity
			handler := middleware.RequireAuth(tt.tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got, _ = middleware.FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/users/me"+tt.query, nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: middleware.AuthCookieName, Value: tt.cookie})
			}
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus != http.StatusOK {
				if called {
					t.Fatal("next handler should not be called")
				}
				return
			}
			if !called {
				t.Fatal("next handler was not called")
			}
			if got.UserID != tt.wantUser {
				t.Fatalf("expected user %q, got %q", tt.wantUser, got.UserID)
			}
		})
	}
}

func TestRequireOrgAccess(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	build := func(tokens fakeAccessTokens) http.Handler {
		r := chi.NewRouter()
		protect := func(next http.Handler) http.Handler {
			return middleware.RequireAuth(tokens)(middleware.RequireOrgAccess(testLogger(), policy)(next))
		}
		ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

		r.With(protect).Get("/expenses", ok)
		r.With(protect).Head("/expenses", ok)
		r.With(protect).Post("/expenses", ok)
		r.With(protect).Get("/projects", ok)
		r.With(protect).Get("/orgs/{orgID}", ok)
		r.With(protect).Get("/unmapped", ok)
		return r
	}

	tests := []struct {
		name       string
		method     string
		path       string
		tokens     fakeAccessTokens
		wantStatus int
		wantOrg    string
	}{
		{
			name:       "missing identity",
			method:     http.MethodGet,
			path:       "/expenses?orgId=org-1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing orgId",
			method:     http.MethodGet,
			path:       "/expenses",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {entities.RoleWildcard}}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no role for org",
			method:     http.MethodGet,
			path:       "/expenses?orgId=org-2",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {entities.RoleWildcard}}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "viewer reads",
			method:     http.MethodGet,
			path:       "/expenses?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {"expense.viewer"}}},
			wantStatus: http.StatusOK,
			wantOrg:    "org-1",
		},
		{
			name:       "viewer head follows get",
			method:     http.MethodHead,
			path:       "/expenses?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {"expense.viewer"}}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "viewer cannot write",
			method:     http.MethodPost,
			path:       "/expenses?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {"expense.viewer"}}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "editor writes",
			method:     http.MethodPost,
			path:       "/expenses?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {"expense.editor"}}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "owner wildcard writes",
			method:     http.MethodPost,
			path:       "/expenses?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {entities.RoleWildcard}}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown role",
			method:     http.MethodGet,
			path:       "/expenses?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {"org.unknown"}}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "path and query mismatch",
			method:     http.MethodGet,
			path:       "/orgs/org-1?orgId=org-2",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {entities.RoleWildcard}}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unmapped route fails closed",
			method:     http.MethodGet,
			path:       "/unmapped?orgId=org-1",
			tokens:     fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {entities.RoleWildcard}}},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.tokens.subject != "" {
				req.Header.Set("Authorization", "Bearer token-1")
			}
			build(tt.tokens).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestRequireOrgAccessSetsIdentity(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	var got middleware.Identity
	r := chi.NewRouter()
	r.With(
		middleware.RequireAuth(fakeAccessTokens{subject: "user-1", orgs: map[string][]string{"org-1": {"expense.editor"}}}),
		middleware.RequireOrgAccess(testLogger(), policy),
	).Get("/expenses", func(_ http.ResponseWriter, r *http.Request) {
		got, _ = middleware.FromContext(r.Context())
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/expenses?orgId=org-1", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got.OrgID != "org-1" || got.UserID != "user-1" {
		t.Fatalf("unexpected identity: %+v", got)
	}
}

type fakeAccessTokens struct {
	subject string
	orgs    map[string][]string
	err     error
}

func (t fakeAccessTokens) ParseAccess(string) (string, map[string][]string, error) {
	if t.err != nil {
		return "", nil, t.err
	}
	return t.subject, t.orgs, nil
}
