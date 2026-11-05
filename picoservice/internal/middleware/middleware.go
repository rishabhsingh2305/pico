// Package middleware provides HTTP middleware for request IDs, structured
// request logging, panic recovery, and query identity checks.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"picoservice/internal/httpx"
)

// RequestIDHeader copies chi's request ID into the X-Request-Id response header
// and the application request context.
func RequestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := chimiddleware.GetReqID(r.Context())
		if requestID == "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set(httpx.HeaderRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(httpx.WithRequestID(r.Context(), requestID)))
	})
}

// RequestLogger logs one structured line per request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				status := ww.Status()
				if status == 0 {
					status = http.StatusOK
				}

				logger.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", chimiddleware.GetReqID(r.Context()),
					"remote_addr", r.RemoteAddr,
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// Recoverer converts a panic into a JSON 500 response when the response has not
// started, and aborts the connection otherwise.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}

				logger.Error("panic recovered",
					"panic", recovered,
					"stack", string(debug.Stack()),
					"request_id", chimiddleware.GetReqID(r.Context()),
				)

				// The response has already started, so a JSON error would be
				// appended to a partial response. Abort it instead.
				if ww.Status() != 0 {
					panic(http.ErrAbortHandler)
				}

				httpx.WriteError(ww, r, http.StatusInternalServerError, "internal server error")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
