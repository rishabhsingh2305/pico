package lambdahttptest

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"picoservice/internal/lambdahttp"
	"picoservice/internal/routes"
	"picoservice/test/fakes"
)

func TestHandleRequestAndResponse(t *testing.T) {
	var received *http.Request

	adapter := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r

		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
		w.Header().Set("X-Response", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"pong"}`))
	}))

	event := baseEvent()
	event.RawQueryString = "foo=bar&baz=qux"
	event.Headers["x-test"] = "value"
	event.Cookies = []string{"a=1", "b=2"}

	resp, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if received == nil {
		t.Fatal("handler was not called")
	}
	if got, want := received.Method, http.MethodGet; got != want {
		t.Fatalf("expected method %q, got %q", want, got)
	}
	if got, want := received.URL.Path, "/ping"; got != want {
		t.Fatalf("expected path %q, got %q", want, got)
	}
	if got, want := received.URL.RawQuery, "foo=bar&baz=qux"; got != want {
		t.Fatalf("expected query %q, got %q", want, got)
	}
	if got, want := received.Header.Get("X-Test"), "value"; got != want {
		t.Fatalf("expected header %q, got %q", want, got)
	}
	if got, want := received.Header.Get("X-Request-Id"), "req-123"; got != want {
		t.Fatalf("expected request id %q, got %q", want, got)
	}
	if got, want := received.RemoteAddr, "203.0.113.10:0"; got != want {
		t.Fatalf("expected remote address %q, got %q", want, got)
	}
	if got, want := len(received.Cookies()), 2; got != want {
		t.Fatalf("expected %d request cookies, got %d", want, got)
	}

	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if got, want := resp.Body, `{"message":"pong"}`; got != want {
		t.Fatalf("expected body %q, got %q", want, got)
	}
	if got, want := resp.Headers["X-Response"], "ok"; got != want {
		t.Fatalf("expected response header %q, got %q", want, got)
	}
	if _, ok := resp.Headers["Set-Cookie"]; ok {
		t.Fatal("expected Set-Cookie to be moved out of response headers")
	}
	if got, want := len(resp.Cookies), 1; got != want {
		t.Fatalf("expected %d response cookies, got %d", want, got)
	}
	if !strings.HasPrefix(resp.Cookies[0], "session=abc") {
		t.Fatalf("expected session cookie, got %q", resp.Cookies[0])
	}
	if resp.IsBase64Encoded {
		t.Fatal("expected a plain-text response body")
	}
}

func TestHandleBase64RequestBody(t *testing.T) {
	var receivedBody string

	adapter := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		receivedBody = string(body)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(body)
	}))

	event := baseEvent()
	event.RequestContext.HTTP.Method = http.MethodPost
	event.Body = base64.StdEncoding.EncodeToString([]byte("hello"))
	event.IsBase64Encoded = true

	resp, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if got, want := receivedBody, "hello"; got != want {
		t.Fatalf("expected request body %q, got %q", want, got)
	}
	if got, want := resp.Body, "hello"; got != want {
		t.Fatalf("expected response body %q, got %q", want, got)
	}
}

func TestHandleInvalidBase64(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	called := false
	adapter := lambdahttp.New(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), logger)

	event := baseEvent()
	event.Body = "not-base64!"
	event.IsBase64Encoded = true

	resp, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if called {
		t.Fatal("handler should not be called for an invalid event")
	}
	if got, want := resp.StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if got, want := resp.Headers["Content-Type"], "application/json; charset=utf-8"; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if !strings.Contains(resp.Body, `"bad request"`) {
		t.Fatalf("expected a bad request error, got %q", resp.Body)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, "convert function url request") {
		t.Fatalf("expected a conversion error log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "request_id=req-123") {
		t.Fatalf("expected a request id in the log, got %q", logOutput)
	}
}

func TestHandlePathFallback(t *testing.T) {
	var receivedPath string

	adapter := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	event := baseEvent()
	event.RawPath = ""
	event.RequestContext.HTTP.Path = "/fallback"

	if _, err := adapter.Handle(context.Background(), event); err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if got, want := receivedPath, "/fallback"; got != want {
		t.Fatalf("expected path %q, got %q", want, got)
	}
}

func TestHandleBinaryResponse(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		wantBody    string
		wantBase64  bool
	}{
		{
			name:        "json",
			contentType: "application/json; charset=utf-8",
			body:        []byte(`{"ok":true}`),
			wantBody:    `{"ok":true}`,
		},
		{
			name:        "text",
			contentType: "text/plain; charset=utf-8",
			body:        []byte("hello"),
			wantBody:    "hello",
		},
		{
			name:        "structured syntax suffix",
			contentType: "application/problem+json",
			body:        []byte(`{"title":"bad"}`),
			wantBody:    `{"title":"bad"}`,
		},
		{
			name:        "binary content type",
			contentType: "application/octet-stream",
			body:        []byte("raw"),
			wantBody:    base64.StdEncoding.EncodeToString([]byte("raw")),
			wantBase64:  true,
		},
		{
			name:        "invalid utf8",
			contentType: "",
			body:        []byte{0xff, 0x00},
			wantBody:    base64.StdEncoding.EncodeToString([]byte{0xff, 0x00}),
			wantBase64:  true,
		},
		{
			name:        "empty binary",
			contentType: "application/octet-stream",
			wantBody:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType := tt.contentType
			body := tt.body
			adapter := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if contentType != "" {
					w.Header().Set("Content-Type", contentType)
				}
				if len(body) > 0 {
					_, _ = w.Write(body)
				}
			}))

			resp, err := adapter.Handle(context.Background(), baseEvent())
			if err != nil {
				t.Fatalf("handle request: %v", err)
			}

			if got := resp.IsBase64Encoded; got != tt.wantBase64 {
				t.Fatalf("expected IsBase64Encoded %t, got %t", tt.wantBase64, got)
			}
			if got := resp.Body; got != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, got)
			}
		})
	}
}

func TestHandleHEADResponse(t *testing.T) {
	adapter := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"message":"pong"}`))
	}))

	event := baseEvent()
	event.RequestContext.HTTP.Method = http.MethodHead

	resp, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if resp.Body != "" {
		t.Fatalf("expected an empty HEAD body, got %q", resp.Body)
	}
	if resp.IsBase64Encoded {
		t.Fatal("expected a non-encoded HEAD response")
	}
}

func TestHandleWithRouter(t *testing.T) {
	logger := testLogger()
	adapter := lambdahttp.New(newTestRouter(t), logger)

	resp, err := adapter.Handle(context.Background(), baseEvent())
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if got, want := strings.TrimSpace(resp.Body), `{"message":"pong"}`; got != want {
		t.Fatalf("expected body %q, got %q", want, got)
	}
	if got, want := resp.Headers["Content-Type"], "application/json; charset=utf-8"; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if got, want := resp.Headers["X-Request-Id"], "req-123"; got != want {
		t.Fatalf("expected request id %q, got %q", want, got)
	}
}

func TestHandleHEADWithRouter(t *testing.T) {
	logger := testLogger()
	adapter := lambdahttp.New(newTestRouter(t), logger)

	event := baseEvent()
	event.RequestContext.HTTP.Method = http.MethodHead

	resp, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
	if resp.Body != "" {
		t.Fatalf("expected an empty HEAD body, got %q", resp.Body)
	}
	if got, want := resp.Headers["Content-Type"], "application/json; charset=utf-8"; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	r, err := routes.New(fakes.NewServices().Deps(testLogger(), stubPinger{}))
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return r
}

func newTestAdapter(handler http.Handler) *lambdahttp.Adapter {
	return lambdahttp.New(handler, testLogger())
}

type stubPinger struct{}

func (stubPinger) Ping(context.Context) error {
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func baseEvent() events.LambdaFunctionURLRequest {
	return events.LambdaFunctionURLRequest{
		Version: "2.0",
		RawPath: "/ping",
		Headers: map[string]string{
			"accept": "application/json",
		},
		RequestContext: events.LambdaFunctionURLRequestContext{
			AccountID:  "123456789012",
			RequestID:  "req-123",
			APIID:      "test-url-id",
			DomainName: "test.lambda-url.us-east-1.on.aws",
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
				Method:   http.MethodGet,
				Path:     "/ping",
				Protocol: "HTTP/1.1",
				SourceIP: "203.0.113.10",
			},
		},
	}
}
