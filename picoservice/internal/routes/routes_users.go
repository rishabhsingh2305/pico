package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func userRoutes(h *controllers.UserController, requireUser func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)

	r.With(requireUser).Get("/me", h.Get)
	r.With(requireUser).Head("/me", h.Get)
	r.With(requireUser).Patch("/me", h.Update)
	r.With(requireUser).Post("/me/password", h.ChangePassword)

	return r
}
