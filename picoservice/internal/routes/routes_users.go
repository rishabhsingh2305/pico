package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func userRoutes(h *controllers.UserController, requireUser func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.With(requireUser).Get("/", h.List)
	r.With(requireUser).Head("/", h.List)

	r.Route("/{userID}", func(r chi.Router) {
		r.With(requireUser).Get("/", h.Get)
		r.With(requireUser).Head("/", h.Get)
		r.With(requireUser).Patch("/", h.Update)
		r.With(requireUser).Post("/password", h.ChangePassword)
	})

	return r
}
