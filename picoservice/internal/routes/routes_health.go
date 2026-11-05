package routes

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func healthRoutes(r chi.Router, logger *slog.Logger, pinger controllers.Pinger) {
	r.Get("/ping", controllers.Ping())
	r.Head("/ping", controllers.Ping())
	r.Get("/ready", controllers.Ready(logger, pinger))
	r.Head("/ready", controllers.Ready(logger, pinger))
}
