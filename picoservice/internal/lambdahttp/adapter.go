// Package lambdahttp adapts Lambda Function URL requests to standard net/http
// handlers.
package lambdahttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"

	"picoservice/internal/httpx"
)

// Adapter bridges Lambda Function URL events and an http.Handler.
type Adapter struct {
	handler http.Handler
	logger  *slog.Logger
}

// New returns an Adapter that serves requests with handler. If logger is nil,
// slog.Default is used.
func New(handler http.Handler, logger *slog.Logger) *Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{handler: handler, logger: logger}
}

// Handle converts a Function URL event into an HTTP request, serves it through
// the wrapped handler, and converts the result back into a Function URL response.
func (a *Adapter) Handle(ctx context.Context, event events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	req, err := toRequest(ctx, event)
	if err != nil {
		a.logger.Error("convert function url request",
			"error", err,
			"request_id", event.RequestContext.RequestID,
		)
		return errorResponse(http.StatusBadRequest, "bad request"), nil
	}

	recorder := newResponseRecorder(req.Method)
	a.handler.ServeHTTP(recorder, req)

	return recorder.response(), nil
}

func toRequest(ctx context.Context, event events.LambdaFunctionURLRequest) (*http.Request, error) {
	body, err := requestBody(event)
	if err != nil {
		return nil, fmt.Errorf("decode request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, requestMethod(event), requestURL(event), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for key, value := range event.Headers {
		req.Header.Add(key, value)
	}
	for _, cookie := range event.Cookies {
		req.Header.Add("Cookie", cookie)
	}

	if sourceIP := event.RequestContext.HTTP.SourceIP; sourceIP != "" {
		req.RemoteAddr = net.JoinHostPort(sourceIP, "0")
	}
	if requestID := event.RequestContext.RequestID; requestID != "" {
		req.Header.Set(httpx.HeaderRequestID, requestID)
	}

	req.RequestURI = req.URL.RequestURI()

	return req, nil
}

func requestBody(event events.LambdaFunctionURLRequest) ([]byte, error) {
	if !event.IsBase64Encoded {
		return []byte(event.Body), nil
	}

	body, err := base64.StdEncoding.DecodeString(event.Body)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	return body, nil
}

func requestURL(event events.LambdaFunctionURLRequest) string {
	path := event.RawPath
	if path == "" {
		path = event.RequestContext.HTTP.Path
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	host := event.RequestContext.DomainName
	if host == "" {
		host = event.Headers["host"]
	}
	if host == "" {
		host = "lambda-url.local"
	}

	rawURL := "https://" + host + path
	if query := strings.TrimPrefix(event.RawQueryString, "?"); query != "" {
		rawURL += "?" + query
	}
	return rawURL
}

func requestMethod(event events.LambdaFunctionURLRequest) string {
	if method := event.RequestContext.HTTP.Method; method != "" {
		return method
	}
	return http.MethodGet
}

type responseRecorder struct {
	header       http.Header
	status       int
	wroteHeader  bool
	body         bytes.Buffer
	suppressBody bool
}

func newResponseRecorder(method string) *responseRecorder {
	return &responseRecorder{
		header:       make(http.Header),
		suppressBody: method == http.MethodHead,
	}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if r.suppressBody {
		return len(p), nil
	}
	return r.body.Write(p)
}

func (r *responseRecorder) response() events.LambdaFunctionURLResponse {
	status := r.status
	if status == 0 {
		status = http.StatusOK
	}

	headers, cookies := splitHeaders(r.header)
	resp := events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    headers,
		Cookies:    cookies,
	}

	// HEAD responses carry headers only, matching net/http behavior.
	if r.suppressBody {
		return resp
	}

	body := r.body.Bytes()
	if shouldBase64Encode(body, r.header.Get(httpx.HeaderContentType)) {
		resp.Body = base64.StdEncoding.EncodeToString(body)
		resp.IsBase64Encoded = true
	} else {
		resp.Body = string(body)
	}

	return resp
}

func splitHeaders(header http.Header) (map[string]string, []string) {
	headers := make(map[string]string, len(header))
	cookies := make([]string, 0, len(header.Values("Set-Cookie")))
	for key, values := range header {
		if strings.EqualFold(key, "Set-Cookie") {
			cookies = append(cookies, values...)
			continue
		}
		headers[key] = strings.Join(values, ",")
	}
	return headers, cookies
}

func shouldBase64Encode(body []byte, contentType string) bool {
	if len(body) == 0 {
		return false
	}
	if !utf8.Valid(body) {
		return true
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return !isTextMediaType(mediaType)
}

func isTextMediaType(mediaType string) bool {
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}

	switch mediaType {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/x-www-form-urlencoded",
		"image/svg+xml":
		return true
	}

	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml")
}

func errorResponse(status int, message string) events.LambdaFunctionURLResponse {
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{httpx.HeaderContentType: httpx.ContentTypeJSON},
		Cookies:    []string{},
		Body:       string(httpx.ErrorBody(message)),
	}
}
