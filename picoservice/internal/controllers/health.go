package controllers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"picoservice/internal/httpx"
)

const readinessTimeout = 3 * time.Second

// Pinger reports whether the database is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Ping responds with a simple liveness payload. It does not depend on the
// database.
func Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, r, http.StatusOK, map[string]string{"message": "pong"})
	}
}

// Ready reports readiness based on a database ping. A failed ping is logged
// with the request ID, but the response never includes the underlying error.
func Ready(logger *slog.Logger, db Pinger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			logger.Error("database readiness check failed",
				"error", err,
				"request_id", httpx.RequestID(r.Context()),
			)
			httpx.WriteError(w, r, http.StatusServiceUnavailable, "database unavailable")
			return
		}

		httpx.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// NotFound responds with a JSON 404 payload.
func NotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, r, http.StatusNotFound, "not found")
	}
}
