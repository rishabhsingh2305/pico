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
	requireOrg func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()

	r.With(requireUser).Get("/", orgs.List)
	r.With(requireUser).Head("/", orgs.List)
	r.With(requireUser).Post("/", orgs.Create)

	r.Route("/{orgID}", func(r chi.Router) {
		r.With(requireOrg).Get("/", orgs.Get)
		r.With(requireOrg).Head("/", orgs.Get)
		r.With(requireOrg).Patch("/", orgs.Update)
		r.Mount("/members", membershipRoutes(members, requireOrg))
	})

	return r
}
