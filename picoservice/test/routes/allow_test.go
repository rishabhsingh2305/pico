package routestest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"picoservice/internal/httpx"
)

func TestAllowHeadersForParameterizedRoutes(t *testing.T) {
	r := newTestRouter(t)

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{name: "users collection", method: http.MethodPut, path: "/users", wantAllow: "POST"},
		{name: "login", method: http.MethodGet, path: "/auth/login", wantAllow: "POST"},
		{name: "logout", method: http.MethodGet, path: "/auth/logout", wantAllow: "POST"},
		{name: "session", method: http.MethodPost, path: "/auth/session", wantAllow: "GET, HEAD"},
		{name: "verify email", method: http.MethodGet, path: "/auth/verify-email", wantAllow: "POST"},
		{name: "verify email otp", method: http.MethodGet, path: "/auth/verify-email/otp", wantAllow: "POST"},
		{name: "resend verification", method: http.MethodGet, path: "/auth/verify-email/resend", wantAllow: "POST"},
		{name: "user me", method: http.MethodPost, path: "/users/me", wantAllow: "GET, HEAD, PATCH"},
		{name: "user password", method: http.MethodDelete, path: "/users/me/password", wantAllow: "POST"},
		{name: "orgs collection", method: http.MethodDelete, path: "/orgs", wantAllow: "GET, HEAD, POST"},
		{name: "org item", method: http.MethodDelete, path: "/orgs/org-1", wantAllow: "GET, HEAD, PATCH"},
		{name: "members collection", method: http.MethodDelete, path: "/orgs/org-1/members", wantAllow: "GET, HEAD, POST"},
		{name: "member item", method: http.MethodGet, path: "/orgs/org-1/members/member-1", wantAllow: "PATCH, DELETE"},
		{name: "projects collection", method: http.MethodDelete, path: "/projects", wantAllow: "GET, HEAD, POST"},
		{name: "project item", method: http.MethodDelete, path: "/projects/project-1", wantAllow: "GET, HEAD, PATCH"},
		{name: "expenses collection", method: http.MethodDelete, path: "/expenses", wantAllow: "GET, HEAD, POST"},
		{name: "expense summary", method: http.MethodPost, path: "/expenses/summary", wantAllow: "GET, HEAD"},
		{name: "expense item", method: http.MethodPost, path: "/expenses/expense-1", wantAllow: "GET, HEAD, PATCH, DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
			if got := rec.Header().Get(httpx.HeaderAllow); got != tt.wantAllow {
				t.Fatalf("expected Allow %q, got %q", tt.wantAllow, got)
			}
		})
	}
}
