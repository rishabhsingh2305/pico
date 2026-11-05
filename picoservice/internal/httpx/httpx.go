// Package httpx provides shared HTTP response and request decoding helpers.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const (
	// HeaderAllow is the HTTP Allow header.
	HeaderAllow = "Allow"
	// HeaderContentType is the HTTP Content-Type header.
	HeaderContentType = "Content-Type"
	// HeaderLocation is the HTTP Location header.
	HeaderLocation = "Location"
	// HeaderRequestID carries the request ID.
	HeaderRequestID = "X-Request-Id"
	// ContentTypeJSON is the content type for JSON responses.
	ContentTypeJSON = "application/json; charset=utf-8"

	maxJSONBodyBytes = 1 << 20
)

type errorPayload struct {
	Error string `json:"error"`
}

type requestIDContextKey struct{}

// WithRequestID returns a copy of ctx carrying requestID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestID returns the request ID carried by ctx, or an empty string.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// WriteJSON writes value as JSON with the given status code. The body is
// omitted for HEAD requests. It encodes the value before writing headers so an
// encoding failure cannot leave a partial response.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeBody(w, r, status, append(body, '\n'))
}

// WriteError writes the standard JSON error envelope with the given status code.
func WriteError(w http.ResponseWriter, r *http.Request, status int, message string) {
	WriteJSON(w, r, status, errorPayload{Error: message})
}

// DecodeJSON decodes a single JSON object from the request body. Unknown fields
// and trailing JSON values are rejected.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeBody(w http.ResponseWriter, r *http.Request, status int, body []byte) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(status)

	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}

// ErrorBody returns the standard JSON error envelope with a trailing newline.
// It is used by transports that cannot write through an http.ResponseWriter.
func ErrorBody(message string) []byte {
	body, err := json.Marshal(errorPayload{Error: message})
	if err != nil {
		return []byte(`{"error":"internal server error"}` + "\n")
	}
	return append(body, '\n')
}
