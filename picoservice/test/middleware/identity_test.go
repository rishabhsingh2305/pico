package middlewaretest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"picoservice/internal/middleware"
)

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		cookie     string
		tokens     fakeAccessTokens
		lookup     fakeUserLookup
		wantStatus int
		wantUser   string
	}{
		{
			name:       "missing identity",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown user",
			query:      "?userId=missing",
			lookup:     fakeUserLookup{existing: map[string]bool{}},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "known user",
			query:      "?userId=user-1",
			lookup:     fakeUserLookup{existing: map[string]bool{"user-1": true}},
			wantStatus: http.StatusOK,
			wantUser:   "user-1",
		},
		{
			name:       "valid cookie",
			cookie:     "token-1",
			tokens:     fakeAccessTokens{subject: "user-1"},
			lookup:     fakeUserLookup{existing: map[string]bool{}},
			wantStatus: http.StatusOK,
			wantUser:   "user-1",
		},
		{
			name:       "invalid cookie does not fall back",
			query:      "?userId=user-1",
			cookie:     "bad-token",
			tokens:     fakeAccessTokens{err: errors.New("invalid token")},
			lookup:     fakeUserLookup{existing: map[string]bool{"user-1": true}},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var got middleware.Identity
			handler := middleware.RequireAuth(testLogger(), tt.lookup, tt.tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got, _ = middleware.FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/users"+tt.query, nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: middleware.AuthCookieName, Value: tt.cookie})
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus != http.StatusOK {
				if called {
					t.Fatal("next handler should not be called")
				}
				return
			}
			if !called {
				t.Fatal("next handler was not called")
			}
			if got.UserID != tt.wantUser {
				t.Fatalf("expected user %q, got %q", tt.wantUser, got.UserID)
			}
		})
	}
}

func TestRequireMember(t *testing.T) {
	tests := []struct {
		name       string
		identity   bool
		query      string
		orgs       fakeOrgLookup
		members    fakeMembershipLookup
		wantStatus int
		wantID     middleware.Identity
	}{
		{
			name:       "missing identity",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing org",
			identity:   true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown org",
			identity:   true,
			query:      "?orgId=org-1",
			orgs:       fakeOrgLookup{existing: map[string]bool{}},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-member",
			identity:   true,
			query:      "?orgId=org-1",
			orgs:       fakeOrgLookup{existing: map[string]bool{"org-1": true}},
			members:    fakeMembershipLookup{existing: map[string]bool{}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "member",
			identity:   true,
			query:      "?orgId=org-1",
			orgs:       fakeOrgLookup{existing: map[string]bool{"org-1": true}},
			members:    fakeMembershipLookup{existing: map[string]bool{"org-1/user-1": true}},
			wantStatus: http.StatusOK,
			wantID:     middleware.Identity{OrgID: "org-1", UserID: "user-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var got middleware.Identity
			handler := middleware.RequireMember(testLogger(), tt.orgs, tt.members)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got, _ = middleware.FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/projects"+tt.query, nil)
			if tt.identity {
				req = req.WithContext(middleware.WithContext(req.Context(), middleware.Identity{UserID: "user-1"}))
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus != http.StatusOK {
				if called {
					t.Fatal("next handler should not be called")
				}
				return
			}
			if !called {
				t.Fatal("next handler was not called")
			}
			if got != tt.wantID {
				t.Fatalf("expected identity %+v, got %+v", tt.wantID, got)
			}
		})
	}
}

func TestLookupErrorIsGeneric(t *testing.T) {
	handler := middleware.RequireAuth(testLogger(), fakeUserLookup{err: errors.New("database down")}, fakeAccessTokens{})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("next handler should not be called")
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users?userId=user-1", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["error"], "internal server error"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

type fakeAccessTokens struct {
	subject string
	err     error
}

func (t fakeAccessTokens) ParseSubject(string) (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return t.subject, nil
}

type fakeUserLookup struct {
	existing map[string]bool
	err      error
}

func (l fakeUserLookup) Exists(_ context.Context, userID string) (bool, error) {
	if l.err != nil {
		return false, l.err
	}
	return l.existing[userID], nil
}

type fakeOrgLookup struct {
	existing map[string]bool
	err      error
}

func (l fakeOrgLookup) Exists(_ context.Context, orgID string) (bool, error) {
	if l.err != nil {
		return false, l.err
	}
	return l.existing[orgID], nil
}

type fakeMembershipLookup struct {
	existing map[string]bool
	err      error
}

func (l fakeMembershipLookup) Exists(_ context.Context, orgID, userID string) (bool, error) {
	if l.err != nil {
		return false, l.err
	}
	return l.existing[orgID+"/"+userID], nil
}
