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

// RequireUser requires a valid userId query parameter, verifies that the user
// exists, and stores the identity in the request context. This is request
// context, not authentication; callers can still supply any user ID until a
// real authentication layer exists.
func RequireUser(logger *slog.Logger, users UserLookup) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := queryID(w, r, UserIDParam)
			if !ok {
				return
			}

			lookupCtx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
			defer cancel()

			exists, err := users.Exists(lookupCtx, userID)
			if err != nil {
				writeLookupError(logger, w, r, "user lookup failed", err)
				return
			}
			if !exists {
				httpx.WriteError(w, r, http.StatusNotFound, "user not found")
				return
			}

			ctx := WithContext(r.Context(), Identity{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireMember requires orgId and userId query parameters, verifies that the
// user and organization exist, and verifies that the user is a member of the
// organization. The identity is stored in the request context.
func RequireMember(logger *slog.Logger, users UserLookup, orgs OrgLookup, memberships MembershipLookup) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := queryID(w, r, OrgIDParam)
			if !ok {
				return
			}
			userID, ok := queryID(w, r, UserIDParam)
			if !ok {
				return
			}

			lookupCtx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
			defer cancel()

			userExists, err := users.Exists(lookupCtx, userID)
			if err != nil {
				writeLookupError(logger, w, r, "user lookup failed", err)
				return
			}
			if !userExists {
				httpx.WriteError(w, r, http.StatusNotFound, "user not found")
				return
			}

			orgExists, err := orgs.Exists(lookupCtx, orgID)
			if err != nil {
				writeLookupError(logger, w, r, "org lookup failed", err)
				return
			}
			if !orgExists {
				httpx.WriteError(w, r, http.StatusNotFound, "org not found")
				return
			}

			member, err := memberships.Exists(lookupCtx, orgID, userID)
			if err != nil {
				writeLookupError(logger, w, r, "membership lookup failed", err)
				return
			}
			if !member {
				httpx.WriteError(w, r, http.StatusForbidden, "forbidden")
				return
			}

			ctx := WithContext(r.Context(), Identity{OrgID: orgID, UserID: userID})
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
