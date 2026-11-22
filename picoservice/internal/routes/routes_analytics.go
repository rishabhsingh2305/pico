package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func analyticsRoutes(
	h *controllers.AnalyticsController,
	linkAnalytics *controllers.LinkAnalyticsController,
	requireOrg func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()

	r.With(requireOrg).Get("/sources", h.List)
	r.With(requireOrg).Head("/sources", h.List)
	r.With(requireOrg).Post("/sources", h.Create)

	r.With(requireOrg).Get("/stats", h.BulkStats)
	r.With(requireOrg).Head("/stats", h.BulkStats)

	// Existing link click stats live in the same analytics namespace.
	r.With(requireOrg).Get("/links", linkAnalytics.Stats)
	r.With(requireOrg).Head("/links", linkAnalytics.Stats)

	r.Route("/sources/{sourceID}", func(r chi.Router) {
		r.With(requireOrg).Get("/", h.Get)
		r.With(requireOrg).Head("/", h.Get)
		r.With(requireOrg).Patch("/", h.Update)
		r.With(requireOrg).Delete("/", h.Delete)
		r.With(requireOrg).Get("/stats", h.Stats)
		r.With(requireOrg).Head("/stats", h.Stats)
	})

	return r
}

func pixelRoutes(h *controllers.AnalyticsPixel) chi.Router {
	r := chi.NewRouter()
	r.Get("/{key}", h.Pixel)
	r.Head("/{key}", h.Pixel)
	return r
}
