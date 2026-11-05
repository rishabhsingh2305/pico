package controllers

import (
	"log/slog"
	"net/http"

	"picoservice/internal/httpx"
)

// logInternalError logs an unexpected service error at the HTTP boundary and
// returns the stable JSON 500 response. Callers must not expose the error.
func logInternalError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, message string, err error) {
	logError(logger, r, message, err)
	httpx.WriteError(w, r, http.StatusInternalServerError, "internal server error")
}

// logError logs an error with request context without writing a response.
func logError(logger *slog.Logger, r *http.Request, message string, err error) {
	logger.Error(message,
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", httpx.RequestID(r.Context()),
	)
}
