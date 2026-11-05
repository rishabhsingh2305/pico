package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func membershipRoutes(h *controllers.MembershipController, requireMember func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireMember).Get("/", h.List)
	r.With(requireMember).Head("/", h.List)
	r.With(requireMember).Post("/", h.Add)
	r.With(requireMember).Delete("/{memberUserID}", h.Remove)

	return r
}
