package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func projectRoutes(h *controllers.ProjectController, requireMember func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireMember).Get("/", h.List)
	r.With(requireMember).Head("/", h.List)
	r.With(requireMember).Post("/", h.Create)

	r.Route("/{projectID}", func(r chi.Router) {
		r.With(requireMember).Get("/", h.Get)
		r.With(requireMember).Head("/", h.Get)
		r.With(requireMember).Patch("/", h.Update)
	})

	return r
}
