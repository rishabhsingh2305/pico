package controllerstest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"picoservice/internal/controllers"
	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestMembershipAdd(t *testing.T) {
	service := &fakes.MembershipService{
		AddFunc: func(_ context.Context, in services.CreateMembershipInput) (entities.Membership, error) {
			if got, want := in.Roles[entities.FeatureExpense], entities.FeatureRoleViewer; got != want {
				t.Fatalf("expected expense role %q, got %q", want, got)
			}
			return entities.Membership{
				ID:        "membership-1",
				OrgID:     in.OrgID,
				UserID:    in.UserID,
				Roles:     in.Roles,
				CreatedAt: time.Now().UTC(),
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/members", strings.NewReader(`{"userId":"user-2","roles":{"expense":"viewer"}}`))
	req = withIdentity(req, "org-1", "owner-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewMembershipController(testLogger(), service).Add(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/orgs/org-1/members/user-2"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
}

func TestMembershipAddConflict(t *testing.T) {
	service := &fakes.MembershipService{
		AddFunc: func(context.Context, services.CreateMembershipInput) (entities.Membership, error) {
			return entities.Membership{}, entities.ErrAlreadyMember
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/members", strings.NewReader(`{"userId":"user-2","roles":{"expense":"viewer"}}`))
	req = withIdentity(req, "org-1", "owner-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewMembershipController(testLogger(), service).Add(rec, req)

	assertJSONError(t, rec, http.StatusConflict, "user is already a member")
}

func TestMembershipList(t *testing.T) {
	service := &fakes.MembershipService{
		ListFunc: func(context.Context, services.ListMembershipsInput) ([]entities.Membership, error) {
			return []entities.Membership{
				{ID: "m-1", OrgID: "org-1", UserID: "owner-1", Owner: true},
				{ID: "m-2", OrgID: "org-1", UserID: "user-2", Roles: map[string]string{entities.FeatureExpense: entities.FeatureRoleEditor}},
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/members", nil)
	req = withIdentity(req, "org-1", "owner-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewMembershipController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"owner":true`) {
		t.Fatalf("expected the owner flag in the response, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"expense":"editor"`) {
		t.Fatalf("expected member roles in the response, got %q", rec.Body.String())
	}
}

func TestMembershipUpdateRoles(t *testing.T) {
	service := &fakes.MembershipService{
		UpdateRolesFunc: func(_ context.Context, orgID, userID string, roles map[string]string) error {
			if orgID != "org-1" || userID != "user-2" {
				t.Fatalf("unexpected update roles input: %q %q", orgID, userID)
			}
			if got, want := roles[entities.FeatureProject], entities.FeatureRoleViewer; got != want {
				t.Fatalf("expected project role %q, got %q", want, got)
			}
			return nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orgs/org-1/members/user-2", strings.NewReader(`{"roles":{"project":"viewer","expense":"editor"}}`))
	req = withIdentity(req, "org-1", "owner-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1", "memberUserID": "user-2"})

	controllers.NewMembershipController(testLogger(), service).UpdateRoles(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestMembershipUpdateRolesOwnerConflict(t *testing.T) {
	service := &fakes.MembershipService{
		UpdateRolesFunc: func(context.Context, string, string, map[string]string) error {
			return entities.ErrCannotChangeOwner
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orgs/org-1/members/owner-1", strings.NewReader(`{"roles":{"project":"viewer"}}`))
	req = withIdentity(req, "org-1", "owner-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1", "memberUserID": "owner-1"})

	controllers.NewMembershipController(testLogger(), service).UpdateRoles(rec, req)

	assertJSONError(t, rec, http.StatusConflict, "cannot change org owner role")
}

func TestMembershipRemove(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := &fakes.MembershipService{
			RemoveFunc: func(_ context.Context, orgID, userID string) error {
				if orgID != "org-1" || userID != "user-2" {
					t.Fatalf("unexpected remove input: %q %q", orgID, userID)
				}
				return nil
			},
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/orgs/org-1/members/user-2", nil)
		req = withIdentity(req, "org-1", "owner-1")
		req = withRouteParams(req, map[string]string{"orgID": "org-1", "memberUserID": "user-2"})

		controllers.NewMembershipController(testLogger(), service).Remove(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("owner conflict", func(t *testing.T) {
		service := &fakes.MembershipService{
			RemoveFunc: func(context.Context, string, string) error {
				return entities.ErrCannotRemoveOwner
			},
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/orgs/org-1/members/owner-1", nil)
		req = withIdentity(req, "org-1", "owner-1")
		req = withRouteParams(req, map[string]string{"orgID": "org-1", "memberUserID": "owner-1"})

		controllers.NewMembershipController(testLogger(), service).Remove(rec, req)

		assertJSONError(t, rec, http.StatusConflict, "cannot remove org owner")
	})
}
