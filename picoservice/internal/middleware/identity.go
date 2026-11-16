package middleware

import (
	"context"
	"net/http"
	"strings"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
)

const (
	// OrgIDParam is the query parameter carrying the organization ID.
	OrgIDParam = "orgId"

	// IdentityKindUser is a request authenticated by a user JWT.
	IdentityKindUser = "user"
	// IdentityKindShare is a request authenticated by a share credential.
	IdentityKindShare = "share"

	maxIDLength = 128
)

// Identity is the authenticated user and, on organization routes, the
// organization and feature roles selected from the token.
type Identity struct {
	UserID  string
	OrgID   string
	Kind    string
	Orgs    map[string][]string
	ShareID string
	Scopes  map[string]struct{}
	Share   *entities.ShareScope
}

// CredentialResolver resolves a raw share token to a credential.
type CredentialResolver interface {
	ResolveToken(ctx context.Context, raw string) (entities.Credential, error)
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
