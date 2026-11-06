package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"picoservice/internal/httpx"
)

const (
	// OrgIDParam is the query parameter carrying the organization ID.
	OrgIDParam = "orgId"
	// UserIDParam is the query parameter carrying the user ID.
	UserIDParam = "userId"

	maxIDLength   = 128
	lookupTimeout = 3 * time.Second
)

// Identity is the organization and user associated with a request.
type Identity struct {
	OrgID  string
	UserID string
}

// UserLookup reports whether a user exists.
type UserLookup interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// OrgLookup reports whether an organization exists.
type OrgLookup interface {
	Exists(ctx context.Context, orgID string) (bool, error)
}

// MembershipLookup reports whether a user belongs to an organization.
type MembershipLookup interface {
	Exists(ctx context.Context, orgID, userID string) (bool, error)
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

// RequireMember requires an authenticated identity and an orgId query
// parameter, verifies that the organization exists, and verifies that the
// identity is a member of the organization. The identity is stored in the
// request context by RequireAuth.
func RequireMember(logger *slog.Logger, orgs OrgLookup, memberships MembershipLookup) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := FromContext(r.Context())
			if !ok || id.UserID == "" {
				httpx.WriteError(w, r, http.StatusUnauthorized, "authentication required")
				return
			}

			orgID, ok := queryID(w, r, OrgIDParam)
			if !ok {
				return
			}

			lookupCtx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
			defer cancel()

			orgExists, err := orgs.Exists(lookupCtx, orgID)
			if err != nil {
				writeLookupError(logger, w, r, "org lookup failed", err)
				return
			}
			if !orgExists {
				httpx.WriteError(w, r, http.StatusNotFound, "org not found")
				return
			}

			member, err := memberships.Exists(lookupCtx, orgID, id.UserID)
			if err != nil {
				writeLookupError(logger, w, r, "membership lookup failed", err)
				return
			}
			if !member {
				httpx.WriteError(w, r, http.StatusForbidden, "forbidden")
				return
			}

			ctx := WithContext(r.Context(), Identity{OrgID: orgID, UserID: id.UserID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

func writeLookupError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, message string, err error) {
	logger.Error(message,
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", httpx.RequestID(r.Context()),
	)
	httpx.WriteError(w, r, http.StatusInternalServerError, "internal server error")
}
