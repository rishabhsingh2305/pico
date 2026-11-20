package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func linkRoutes(h *controllers.LinkController, requireOrg func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireOrg).Get("/", h.List)
	r.With(requireOrg).Head("/", h.List)
	r.With(requireOrg).Post("/", h.Create)

	r.Route("/{linkID}", func(r chi.Router) {
		r.With(requireOrg).Get("/", h.Get)
		r.With(requireOrg).Head("/", h.Get)
		r.With(requireOrg).Patch("/", h.Update)
		r.With(requireOrg).Delete("/", h.Delete)
	})

	return r
}

func redirectRoutes(h *controllers.LinkController) chi.Router {
	r := chi.NewRouter()

	r.Get("/{slug}", h.Redirect)
	r.Head("/{slug}", h.Redirect)

	return r
}
