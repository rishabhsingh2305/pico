package routestest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/middleware"
	"picoservice/internal/routes"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestRoutes(t *testing.T) {
	r := newTestRouter(t)

	tests := []struct {
		name          string
		method        string
		path          string
		requestID     string
		wantStatus    int
		wantBody      map[string]string
		wantEmptyBody bool
		wantAllow     string
	}{
		{
			name:       "ping",
			method:     http.MethodGet,
			path:       "/ping",
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"message": "pong"},
		},
		{
			name:          "ping head",
			method:        http.MethodHead,
			path:          "/ping",
			wantStatus:    http.StatusOK,
			wantEmptyBody: true,
		},
		{
			name:       "ready",
			method:     http.MethodGet,
			path:       "/ready",
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "ok"},
		},
		{
			name:          "ready head",
			method:        http.MethodHead,
			path:          "/ready",
			wantStatus:    http.StatusOK,
			wantEmptyBody: true,
		},
		{
			name:       "not found",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
			wantBody:   map[string]string{"error": "not found"},
		},
		{
			name:       "ping method not allowed",
			method:     http.MethodPost,
			path:       "/ping",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   map[string]string{"error": "method not allowed"},
			wantAllow:  "GET, HEAD",
		},
		{
			name:       "ready method not allowed",
			method:     http.MethodPost,
			path:       "/ready",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   map[string]string{"error": "method not allowed"},
			wantAllow:  "GET, HEAD",
		},
		{
			name:       "preserves incoming request id",
			method:     http.MethodGet,
			path:       "/ping",
			requestID:  "test-request-id",
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"message": "pong"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.requestID != "" {
				req.Header.Set(httpx.HeaderRequestID, tt.requestID)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if got := rec.Header().Get(httpx.HeaderContentType); !strings.HasPrefix(got, "application/json") {
				t.Fatalf("expected JSON content type, got %q", got)
			}
			if requestID := rec.Header().Get(httpx.HeaderRequestID); requestID == "" {
				t.Fatal("expected X-Request-Id response header")
			} else if tt.requestID != "" && requestID != tt.requestID {
				t.Fatalf("expected request id %q, got %q", tt.requestID, requestID)
			}
			if got := rec.Header().Get(httpx.HeaderAllow); got != tt.wantAllow {
				t.Fatalf("expected Allow header %q, got %q", tt.wantAllow, got)
			}

			if tt.wantEmptyBody {
				if rec.Body.Len() != 0 {
					t.Fatalf("expected an empty body, got %q", rec.Body.String())
				}
				return
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !maps.Equal(body, tt.wantBody) {
				t.Fatalf("expected body %v, got %v", tt.wantBody, body)
			}
		})
	}
}

func TestReadyUnavailable(t *testing.T) {
	r := newRouter(t, fakes.NewServices(), stubPinger{err: errors.New("connection refused")})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "database unavailable"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestRequestIDContext(t *testing.T) {
	r := newRouter(t, fakes.NewServices(), stubPinger{})
	r.Get("/request-id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(httpx.RequestID(r.Context())))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	req.Header.Set(httpx.HeaderRequestID, "test-request-id")

	r.ServeHTTP(rec, req)

	if got, want := rec.Body.String(), "test-request-id"; got != want {
		t.Fatalf("expected request id %q, got %q", want, got)
	}
}

func TestPanicRecovery(t *testing.T) {
	r := newRouter(t, fakes.NewServices(), stubPinger{})
	r.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if rec.Header().Get(httpx.HeaderRequestID) == "" {
		t.Fatal("expected X-Request-Id response header")
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "internal server error"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestPanicRecoveryAbortsStartedResponse(t *testing.T) {
	r := newRouter(t, fakes.NewServices(), stubPinger{})
	r.Get("/panic", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		panic("boom")
	})

	assertAbortPanic(t, func() {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))
	})
}

func TestPanicRecoveryRepanicsAbortHandler(t *testing.T) {
	r := newRouter(t, fakes.NewServices(), stubPinger{})
	r.Get("/abort", func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	})

	assertAbortPanic(t, func() {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/abort", nil))
	})
}

func TestForwardedForIsIgnored(t *testing.T) {
	r := newRouter(t, fakes.NewServices(), stubPinger{})
	r.Get("/remote", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.RemoteAddr))
	})

	req := httptest.NewRequest(http.MethodGet, "/remote", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if got, want := rec.Body.String(), "192.0.2.1:1234"; got != want {
		t.Fatalf("expected remote address %q, got %q", want, got)
	}
}

func TestNewRequiresDependencies(t *testing.T) {
	if _, err := routes.New(routes.Deps{}); err == nil {
		t.Fatal("expected an error for missing dependencies")
	}
}

func TestAllowHeadersForParameterizedRoutes(t *testing.T) {
	r := newTestRouter(t)

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{name: "users collection", method: http.MethodPut, path: "/users", wantAllow: "GET, HEAD, POST"},
		{name: "login", method: http.MethodGet, path: "/auth/login", wantAllow: "POST"},
		{name: "logout", method: http.MethodGet, path: "/auth/logout", wantAllow: "POST"},
		{name: "session", method: http.MethodPost, path: "/auth/session", wantAllow: "GET, HEAD"},
		{name: "verify email", method: http.MethodGet, path: "/auth/verify-email", wantAllow: "POST"},
		{name: "resend verification", method: http.MethodGet, path: "/auth/verify-email/resend", wantAllow: "POST"},
		{name: "user item", method: http.MethodPost, path: "/users/user-1", wantAllow: "GET, HEAD, PATCH"},
		{name: "user password", method: http.MethodDelete, path: "/users/user-1/password", wantAllow: "POST"},
		{name: "orgs collection", method: http.MethodDelete, path: "/orgs", wantAllow: "GET, HEAD, POST"},
		{name: "org item", method: http.MethodDelete, path: "/orgs/org-1", wantAllow: "GET, HEAD, PATCH"},
		{name: "members collection", method: http.MethodDelete, path: "/orgs/org-1/members", wantAllow: "GET, HEAD, POST"},
		{name: "member item", method: http.MethodGet, path: "/orgs/org-1/members/member-1", wantAllow: "DELETE"},
		{name: "projects collection", method: http.MethodDelete, path: "/projects", wantAllow: "GET, HEAD, POST"},
		{name: "project item", method: http.MethodDelete, path: "/projects/project-1", wantAllow: "GET, HEAD, PATCH"},
		{name: "expenses collection", method: http.MethodDelete, path: "/expenses", wantAllow: "GET, HEAD, POST"},
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

func TestVerificationRoutes(t *testing.T) {
	r := newTestRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":"token-1"}`))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/verify-email/resend", strings.NewReader(`{"email":"ada@example.com"}`))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestAuthRoutes(t *testing.T) {
	r := newTestRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","password":"correct horse"}`))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), middleware.AuthCookieName) {
		t.Fatalf("expected auth cookie, got %q", rec.Header().Get("Set-Cookie"))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AuthCookieName, Value: "token-1"})
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestRequireAuthOnRoutes(t *testing.T) {
	fakeServices := fakes.NewServices()
	fakeServices.Users.Existing = map[string]bool{"user-1": true}
	r := newRouter(t, fakeServices, stubPinger{})

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "missing", path: "/users", wantStatus: http.StatusUnauthorized},
		{name: "unknown", path: "/users?userId=missing", wantStatus: http.StatusNotFound},
		{name: "known", path: "/users?userId=user-1", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestRequireMemberOnProjects(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		users      map[string]bool
		orgs       map[string]bool
		members    map[string]bool
		wantStatus int
	}{
		{
			name:       "missing identity",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown user",
			query:      "?orgId=org-1&userId=user-1",
			users:      map[string]bool{},
			orgs:       map[string]bool{"org-1": true},
			members:    map[string]bool{"org-1/user-1": true},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown org",
			query:      "?orgId=org-1&userId=user-1",
			users:      map[string]bool{"user-1": true},
			orgs:       map[string]bool{},
			members:    map[string]bool{"org-1/user-1": true},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-member",
			query:      "?orgId=org-1&userId=user-1",
			users:      map[string]bool{"user-1": true},
			orgs:       map[string]bool{"org-1": true},
			members:    map[string]bool{},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "member",
			query:      "?orgId=org-1&userId=user-1",
			users:      map[string]bool{"user-1": true},
			orgs:       map[string]bool{"org-1": true},
			members:    map[string]bool{"org-1/user-1": true},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServices := fakes.NewServices()
			fakeServices.Users.Existing = tt.users
			fakeServices.Orgs.Existing = tt.orgs
			fakeServices.Memberships.Existing = tt.members
			r := newRouter(t, fakeServices, stubPinger{})

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/projects"+tt.query, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestCreateOrgRoute(t *testing.T) {
	fakeServices := fakes.NewServices()
	fakeServices.Users.Existing = map[string]bool{"user-1": true}
	fakeServices.Orgs.CreateFunc = func(_ context.Context, in services.CreateOrgInput) (entities.Org, error) {
		if in.ActorID != "user-1" {
			t.Fatalf("expected actor user-1, got %q", in.ActorID)
		}
		return entities.Org{ID: "org-1", Name: in.Name, UserID: in.ActorID}, nil
	}
	r := newRouter(t, fakeServices, stubPinger{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs?userId=user-1", strings.NewReader(`{"name":"Acme"}`))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/orgs/org-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
}

func assertAbortPanic(t *testing.T, fn func()) {
	t.Helper()

	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("expected a panic")
			}
			if recovered != http.ErrAbortHandler {
				t.Fatalf("expected %v, got %v", http.ErrAbortHandler, recovered)
			}
		}()

		fn()
	}()
}

type stubPinger struct {
	err error
}

func (p stubPinger) Ping(context.Context) error {
	return p.err
}

func newTestRouter(t *testing.T) *chi.Mux {
	t.Helper()
	return newRouter(t, fakes.NewServices(), stubPinger{})
}

func newRouter(t *testing.T, fakeServices fakes.Services, pinger stubPinger) *chi.Mux {
	t.Helper()

	r, err := routes.New(fakeServices.Deps(testLogger(), pinger))
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return r
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
