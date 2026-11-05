package httpxtest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"picoservice/internal/httpx"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	httpx.WriteJSON(rec, req, http.StatusCreated, map[string]string{"message": "pong"})

	if got, want := rec.Code, http.StatusCreated; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if got, want := rec.Body.String(), `{"message":"pong"}`+"\n"; got != want {
		t.Fatalf("expected body %q, got %q", want, got)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	httpx.WriteError(rec, req, http.StatusNotFound, "not found")

	if got, want := rec.Code, http.StatusNotFound; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "not found"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestWriteJSONHead(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/", nil)

	httpx.WriteJSON(rec, req, http.StatusOK, map[string]string{"message": "pong"})

	if got, want := rec.Code, http.StatusOK; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body, got %q", rec.Body.String())
	}
}

func TestWriteErrorHead(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/", nil)

	httpx.WriteError(rec, req, http.StatusNotFound, "not found")

	if got, want := rec.Code, http.StatusNotFound; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if got, want := rec.Header().Get(httpx.HeaderContentType), httpx.ContentTypeJSON; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body, got %q", rec.Body.String())
	}
}

func TestWriteJSONEncodingFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	httpx.WriteJSON(rec, req, http.StatusOK, make(chan int))

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
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
}

func TestRequestID(t *testing.T) {
	ctx := httpx.WithRequestID(context.Background(), "test-request-id")

	if got, want := httpx.RequestID(ctx), "test-request-id"; got != want {
		t.Fatalf("expected request id %q, got %q", want, got)
	}
	if got := httpx.RequestID(context.Background()); got != "" {
		t.Fatalf("expected an empty request id, got %q", got)
	}
}

func TestErrorBody(t *testing.T) {
	if got, want := string(httpx.ErrorBody("bad request")), `{"error":"bad request"}`+"\n"; got != want {
		t.Fatalf("expected error body %q, got %q", want, got)
	}
}
