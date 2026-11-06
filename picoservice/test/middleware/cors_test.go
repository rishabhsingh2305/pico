package middlewaretest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"picoservice/internal/middleware"
)

func TestCORS(t *testing.T) {
	const allowedOrigin = "http://localhost:5173"

	tests := []struct {
		name             string
		origins          []string
		method           string
		origin           string
		preflight        bool
		wantStatus       int
		wantOrigin       string
		wantCredentials  string
		wantAllowMethods string
		wantNextCalled   bool
	}{
		{
			name:            "allowed origin",
			origins:         []string{allowedOrigin},
			method:          http.MethodGet,
			origin:          allowedOrigin,
			wantStatus:      http.StatusOK,
			wantOrigin:      allowedOrigin,
			wantCredentials: "true",
			wantNextCalled:  true,
		},
		{
			name:            "multiple allowed origins",
			origins:         []string{allowedOrigin, "https://app.example.com"},
			method:          http.MethodGet,
			origin:          "https://app.example.com",
			wantStatus:      http.StatusOK,
			wantOrigin:      "https://app.example.com",
			wantCredentials: "true",
			wantNextCalled:  true,
		},
		{
			name:           "disallowed origin",
			origins:        []string{allowedOrigin},
			method:         http.MethodGet,
			origin:         "http://evil.example",
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:             "preflight",
			origins:          []string{allowedOrigin},
			method:           http.MethodOptions,
			origin:           allowedOrigin,
			preflight:        true,
			wantStatus:       http.StatusNoContent,
			wantOrigin:       allowedOrigin,
			wantCredentials:  "true",
			wantAllowMethods: "GET, HEAD, POST, PATCH, DELETE, OPTIONS",
		},
		{
			name:            "allow all reflects origin",
			origins:         []string{"*"},
			method:          http.MethodGet,
			origin:          "https://any.example",
			wantStatus:      http.StatusOK,
			wantOrigin:      "https://any.example",
			wantCredentials: "true",
			wantNextCalled:  true,
		},
		{
			name:           "allow all rejects null origin",
			origins:        []string{"*"},
			method:         http.MethodGet,
			origin:         "null",
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "no origin",
			origins:        []string{allowedOrigin},
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := middleware.CORS(tt.origins)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, "/auth/login", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.preflight {
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Fatalf("expected allow origin %q, got %q", tt.wantOrigin, got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != tt.wantCredentials {
				t.Fatalf("expected allow credentials %q, got %q", tt.wantCredentials, got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Methods"); got != tt.wantAllowMethods {
				t.Fatalf("expected allow methods %q, got %q", tt.wantAllowMethods, got)
			}
			if called != tt.wantNextCalled {
				t.Fatalf("expected next called %v, got %v", tt.wantNextCalled, called)
			}
		})
	}
}
