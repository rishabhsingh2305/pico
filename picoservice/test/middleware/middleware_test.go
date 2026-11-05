package middlewaretest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"picoservice/internal/httpx"
	"picoservice/internal/middleware"
)

func TestRequestIDHeader(t *testing.T) {
	var got string
	handler := middleware.RequestIDHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = httpx.RequestID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimiddleware.RequestIDKey, "request-1"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if want := "request-1"; got != want {
		t.Fatalf("expected context request id %q, got %q", want, got)
	}
	if want := "request-1"; rec.Header().Get(httpx.HeaderRequestID) != want {
		t.Fatalf("expected response request id %q, got %q", want, rec.Header().Get(httpx.HeaderRequestID))
	}
}

func TestRequestIDHeaderWithoutID(t *testing.T) {
	called := false
	handler := middleware.RequestIDHeader(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("expected the next handler to be called")
	}
	if got := rec.Header().Get(httpx.HeaderRequestID); got != "" {
		t.Fatalf("expected no request id header, got %q", got)
	}
}

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/things", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimiddleware.RequestIDKey, "request-1"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}

	checks := map[string]any{
		"msg":        "request",
		"method":     http.MethodPost,
		"path":       "/things",
		"status":     float64(http.StatusCreated),
		"bytes":      float64(2),
		"request_id": "request-1",
	}
	for key, want := range checks {
		if got := entry[key]; got != want {
			t.Fatalf("expected log field %q to be %v, got %v", key, want, got)
		}
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Fatal("expected duration_ms log field")
	}
}

func TestRecoverer(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.Recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "internal server error"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}
	if got, want := entry["msg"], "panic recovered"; got != want {
		t.Fatalf("expected log message %q, got %v", want, got)
	}
	if got, want := entry["panic"], "boom"; got != want {
		t.Fatalf("expected logged panic %q, got %v", want, got)
	}
	if stack, ok := entry["stack"].(string); !ok || stack == "" {
		t.Fatal("expected non-empty stack log field")
	}
}

func TestRecovererAbortsStartedResponse(t *testing.T) {
	handler := middleware.Recoverer(testLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		panic("boom")
	}))

	assertAbortPanic(t, func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

func TestRecovererRepanicsAbortHandler(t *testing.T) {
	handler := middleware.Recoverer(testLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	assertAbortPanic(t, func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
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

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
