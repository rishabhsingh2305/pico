package controllers

import (
	"net/http"

	"picoservice/internal/httpx"
	"picoservice/internal/middleware"
)

func requestIdentity(w http.ResponseWriter, r *http.Request) (middleware.Identity, bool) {
	id, ok := middleware.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, http.StatusInternalServerError, "internal server error")
	}
	return id, ok
}

func isShareIdentity(id middleware.Identity) bool {
	return id.Kind == middleware.IdentityKindShare && id.Share != nil
}
