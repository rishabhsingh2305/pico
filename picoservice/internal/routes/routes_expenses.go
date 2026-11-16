package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func expenseRoutes(h *controllers.ExpenseController, requireOrg func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireOrg).Get("/", h.List)
	r.With(requireOrg).Head("/", h.List)
	r.With(requireOrg).Post("/", h.Create)

	r.With(requireOrg).Get("/summary", h.Summarize)
	r.With(requireOrg).Head("/summary", h.Summarize)

	r.Route("/{expenseID}", func(r chi.Router) {
		r.With(requireOrg).Get("/", h.Get)
		r.With(requireOrg).Head("/", h.Get)
		r.With(requireOrg).Patch("/", h.Update)
		r.With(requireOrg).Delete("/", h.Delete)
	})

	return r
}
