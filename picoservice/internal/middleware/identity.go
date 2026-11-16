package middleware

import (
	"context"
	"net/http"
	"strings"

	"picoservice/internal/httpx"
)

const (
	// OrgIDParam is the query parameter carrying the organization ID.
	OrgIDParam = "orgId"

	maxIDLength = 128
)

// Identity is the authenticated user and, on organization routes, the
// organization and feature roles selected from the token.
type Identity struct {
	UserID string
	OrgID  string
	Orgs   map[string][]string
}

type contextKey struct{}

// WithContext returns a copy of ctx carrying id.
func WithContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the identity carried by ctx, if any.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(contextKey{}).(Identity)
	return id, ok
}

func queryID(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	id := strings.TrimSpace(r.URL.Query().Get(key))
	if id == "" {
		httpx.WriteError(w, r, http.StatusBadRequest, key+" is required")
		return "", false
	}
	if len(id) > maxIDLength {
		httpx.WriteError(w, r, http.StatusBadRequest, key+" must be at most 128 characters")
		return "", false
	}
	return id, true
}
