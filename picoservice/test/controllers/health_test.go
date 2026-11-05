package controllerstest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"picoservice/internal/controllers"
	"picoservice/internal/httpx"
)

func TestPing(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantBody string
	}{
		{
			name:     "get",
			method:   http.MethodGet,
			wantBody: `{"message":"pong"}` + "\n",
		},
		{
			name:     "head",
			method:   http.MethodHead,
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/ping", nil)

			controllers.Ping().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}
			if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
				t.Fatalf("expected content type %q, got %q", want, got)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, got)
			}
		})
	}
}

func TestReady(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		pingErr    error
		wantStatus int
		wantBody   map[string]string
	}{
		{
			name:       "ready",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "ok"},
		},
		{
			name:       "ready head",
			method:     http.MethodHead,
			wantStatus: http.StatusOK,
		},
		{
			name:       "database unavailable",
			method:     http.MethodGet,
			pingErr:    errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   map[string]string{"error": "database unavailable"},
		},
		{
			name:       "database unavailable head",
			method:     http.MethodHead,
			pingErr:    errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/ready", nil)
			handler := controllers.Ready(testLogger(), stubPinger{err: tt.pingErr})

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
				t.Fatalf("expected content type %q, got %q", want, got)
			}

			if tt.method == http.MethodHead {
				if rec.Body.Len() != 0 {
					t.Fatalf("expected an empty body, got %q", rec.Body.String())
				}
				return
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got, want := body["status"], tt.wantBody["status"]; got != want {
				t.Fatalf("expected status field %q, got %q", want, got)
			}
			if got, want := body["error"], tt.wantBody["error"]; got != want {
				t.Fatalf("expected error field %q, got %q", want, got)
			}
		})
	}
}

func TestReadyAppliesTimeout(t *testing.T) {
	pinger := &deadlinePinger{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	controllers.Ready(testLogger(), pinger).ServeHTTP(rec, req)

	if !pinger.hasDeadline {
		t.Fatal("expected the readiness ping to receive a deadline")
	}
}

func TestReadyLogsPingError(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	req = req.WithContext(httpx.WithRequestID(req.Context(), "test-request-id"))
	rec := httptest.NewRecorder()

	controllers.Ready(logger, stubPinger{err: errors.New("connection refused")}).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("response leaked the database error: %q", rec.Body.String())
	}
	for _, want := range []string{"connection refused", "test-request-id"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("expected logs to contain %q, got %q", want, logs.String())
		}
	}
}

func TestNotFound(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "get",
			method:     http.MethodGet,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "head",
			method:     http.MethodHead,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/unknown", nil)

			controllers.NotFound().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
				t.Fatalf("expected content type %q, got %q", want, got)
			}
			if tt.method == http.MethodHead && rec.Body.Len() != 0 {
				t.Fatalf("expected an empty body, got %q", rec.Body.String())
			}
		})
	}
}

type stubPinger struct {
	err error
}

func (p stubPinger) Ping(context.Context) error {
	return p.err
}

type deadlinePinger struct {
	hasDeadline bool
}

func (p *deadlinePinger) Ping(ctx context.Context) error {
	_, p.hasDeadline = ctx.Deadline()
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("expected status %d, got %d", status, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body["error"]; got != message {
		t.Fatalf("expected error %q, got %q", message, got)
	}
}
