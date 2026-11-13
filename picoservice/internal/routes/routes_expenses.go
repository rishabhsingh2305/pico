package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func expenseRoutes(h *controllers.ExpenseController, requireMember func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(requireMember).Get("/", h.List)
	r.With(requireMember).Head("/", h.List)
	r.With(requireMember).Post("/", h.Create)

	r.With(requireMember).Get("/summary", h.Summarize)
	r.With(requireMember).Head("/summary", h.Summarize)

	r.Route("/{expenseID}", func(r chi.Router) {
		r.With(requireMember).Get("/", h.Get)
		r.With(requireMember).Head("/", h.Get)
		r.With(requireMember).Patch("/", h.Update)
		r.With(requireMember).Delete("/", h.Delete)
	})

	return r
}
