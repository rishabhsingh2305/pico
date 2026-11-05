package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func orgRoutes(
	orgs *controllers.OrgController,
	members *controllers.MembershipController,
	requireUser func(http.Handler) http.Handler,
	requireMember func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()

	r.With(requireUser).Get("/", orgs.List)
	r.With(requireUser).Head("/", orgs.List)
	r.With(requireUser).Post("/", orgs.Create)

	r.Route("/{orgID}", func(r chi.Router) {
		r.With(requireMember).Get("/", orgs.Get)
		r.With(requireMember).Head("/", orgs.Get)
		r.With(requireMember).Patch("/", orgs.Update)
		r.Mount("/members", membershipRoutes(members, requireMember))
	})

	return r
}
