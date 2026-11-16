package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func membershipRoutes(h *controllers.MembershipController, requireOrg func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireOrg).Get("/", h.List)
	r.With(requireOrg).Head("/", h.List)
	r.With(requireOrg).Post("/", h.Add)
	r.With(requireOrg).Patch("/{memberUserID}", h.UpdateRoles)
	r.With(requireOrg).Delete("/{memberUserID}", h.Remove)

	return r
}
