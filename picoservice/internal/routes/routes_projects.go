package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func projectRoutes(h *controllers.ProjectController, requireOrg func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireOrg).Get("/", h.List)
	r.With(requireOrg).Head("/", h.List)
	r.With(requireOrg).Post("/", h.Create)

	r.Route("/{projectID}", func(r chi.Router) {
		r.With(requireOrg).Get("/", h.Get)
		r.With(requireOrg).Head("/", h.Get)
		r.With(requireOrg).Patch("/", h.Update)
	})

	return r
}
