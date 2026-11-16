package membershipstest

import (
	"context"
	"errors"
	"testing"
	"time"

	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestAddMember(t *testing.T) {
	store := fakes.NewMembershipDAO()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users, fakeFeatures{})

	membership, err := service.Add(context.Background(), services.CreateMembershipInput{
		OrgID:  "org-1",
		UserID: "user-2",
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if got, want := membership.Roles[entities.FeatureExpense], entities.FeatureRoleEditor; got != want {
		t.Fatalf("expected expense role %q, got %q", want, got)
	}
	if !store.Has("org-1", "user-2") {
		t.Fatal("expected membership to be stored")
	}
}

func TestAddRejectsMissingUser(t *testing.T) {
	store := fakes.NewMembershipDAO()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{}
	service := services.NewMembershipService(store, orgs, users, fakeFeatures{})

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		OrgID:  "org-1",
		UserID: "missing",
	})
	if !errors.Is(err, entities.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	store := fakes.NewMembershipDAO()
	store.Add("org-1", "user-2", nil)
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users, fakeFeatures{})

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		OrgID:  "org-1",
		UserID: "user-2",
	})
	if !errors.Is(err, entities.ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestAddMemberWithRoles(t *testing.T) {
	store := fakes.NewMembershipDAO()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users, fakeFeatures{})

	membership, err := service.Add(context.Background(), services.CreateMembershipInput{
		OrgID:  "org-1",
		UserID: "user-2",
		Roles: map[string]string{
			entities.FeatureProject: entities.FeatureRoleViewer,
			entities.FeatureExpense: entities.FeatureRoleViewer,
		},
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if got, want := membership.Roles[entities.FeatureProject], entities.FeatureRoleViewer; got != want {
		t.Fatalf("expected project role %q, got %q", want, got)
	}
	if _, ok := membership.Roles[entities.FeatureOrg]; ok {
		t.Fatal("expected omitted org role to stay omitted")
	}
}

func TestAddRejectsInvalidFeatureRole(t *testing.T) {
	store := fakes.NewMembershipDAO()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users, fakeFeatures{})

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		OrgID:  "org-1",
		UserID: "user-2",
		Roles:  map[string]string{entities.FeatureMember: entities.FeatureRoleEditor},
	})
	if err == nil {
		t.Fatal("expected a validation error for member.editor")
	}
}

func TestAddRejectsEmptyRoles(t *testing.T) {
	store := fakes.NewMembershipDAO()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users, fakeFeatures{})

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		OrgID:  "org-1",
		UserID: "user-2",
		Roles:  map[string]string{},
	})
	if err == nil {
		t.Fatal("expected a validation error for empty roles")
	}
}

func TestUpdateRolesRequiresRoles(t *testing.T) {
	store := fakes.NewMembershipDAO()
	store.Add("org-1", "user-2", nil)
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

	if err := service.UpdateRoles(context.Background(), "org-1", "user-2", nil); err == nil {
		t.Fatal("expected a validation error for a missing roles map")
	}
}

func TestLegacyMemberRoles(t *testing.T) {
	editor := entities.LegacyMemberRoles("org.editor")
	if got, want := editor[entities.FeatureExpense], entities.FeatureRoleEditor; got != want {
		t.Fatalf("expected expense role %q, got %q", want, got)
	}
	if got, want := editor[entities.FeatureMember], entities.FeatureRoleViewer; got != want {
		t.Fatalf("expected member role %q, got %q", want, got)
	}

	viewer := entities.LegacyMemberRoles("org.viewer")
	if got, want := viewer[entities.FeatureExpense], entities.FeatureRoleViewer; got != want {
		t.Fatalf("expected expense role %q, got %q", want, got)
	}
	if _, ok := viewer[entities.FeatureMember]; ok {
		t.Fatal("legacy viewer must not gain member access")
	}
}

func TestUpdateRoles(t *testing.T) {
	t.Run("owner changes roles", func(t *testing.T) {
		store := fakes.NewMembershipDAO()
		store.Add("org-1", "user-2", nil)
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

		roles := map[string]string{
			entities.FeatureProject: entities.FeatureRoleViewer,
			entities.FeatureExpense: entities.FeatureRoleAdmin,
		}
		if err := service.UpdateRoles(context.Background(), "org-1", "user-2", roles); err != nil {
			t.Fatalf("update roles: %v", err)
		}
		if got, want := store.Memberships["org-1/user-2"].Roles[entities.FeatureExpense], entities.FeatureRoleAdmin; got != want {
			t.Fatalf("expected expense role %q, got %q", want, got)
		}
	})

	t.Run("owner roles cannot change", func(t *testing.T) {
		store := fakes.NewMembershipDAO()
		store.Add("org-1", "owner-1", nil)
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

		err := service.UpdateRoles(context.Background(), "org-1", "owner-1", entities.ViewerRoles())
		if !errors.Is(err, entities.ErrCannotChangeOwner) {
			t.Fatalf("expected ErrCannotChangeOwner, got %v", err)
		}
	})

	t.Run("unknown role", func(t *testing.T) {
		store := fakes.NewMembershipDAO()
		store.Add("org-1", "user-2", nil)
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

		err := service.UpdateRoles(context.Background(), "org-1", "user-2", map[string]string{
			entities.FeatureExpense: "owner",
		})
		if err == nil {
			t.Fatal("expected a validation error")
		}
	})
}

func TestListMarksOwner(t *testing.T) {
	store := fakes.NewMembershipDAO()
	store.AddWithTime("org-1", "owner-1", nil, time.Now().UTC())
	store.AddWithTime("org-1", "user-2", nil, time.Now().UTC().Add(-time.Minute))
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

	items, err := service.List(context.Background(), services.ListMembershipsInput{
		OrgID: "org-1",
	})
	if err != nil {
		t.Fatalf("list memberships: %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("expected %d memberships, got %d", want, got)
	}

	for _, item := range items {
		switch item.UserID {
		case "owner-1":
			if !item.Owner {
				t.Fatal("expected the owner flag on the owner membership")
			}
		case "user-2":
			if item.Owner {
				t.Fatal("did not expect the owner flag on a member")
			}
			if got, want := item.Roles[entities.FeatureExpense], entities.FeatureRoleEditor; got != want {
				t.Fatalf("expected expense role %q, got %q", want, got)
			}
		}
	}
}

func TestListMissingOrg(t *testing.T) {
	service := services.NewMembershipService(fakes.NewMembershipDAO(), &fakeOrgs{}, &fakeUsers{}, fakeFeatures{})

	_, err := service.List(context.Background(), services.ListMembershipsInput{OrgID: "missing"})
	if !errors.Is(err, entities.ErrOrgNotFound) {
		t.Fatalf("expected ErrOrgNotFound, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	t.Run("owner cannot remove self", func(t *testing.T) {
		store := fakes.NewMembershipDAO()
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

		err := service.Remove(context.Background(), "org-1", "owner-1")
		if !errors.Is(err, entities.ErrCannotRemoveOwner) {
			t.Fatalf("expected ErrCannotRemoveOwner, got %v", err)
		}
	})

	t.Run("owner removes member", func(t *testing.T) {
		store := fakes.NewMembershipDAO()
		store.Add("org-1", "user-2", nil)
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{}, fakeFeatures{})

		if err := service.Remove(context.Background(), "org-1", "user-2"); err != nil {
			t.Fatalf("remove member: %v", err)
		}
		if store.Has("org-1", "user-2") {
			t.Fatal("expected membership to be removed")
		}
	})
}

func TestAddOwner(t *testing.T) {
	store := fakes.NewMembershipDAO()
	service := services.NewMembershipService(store, &fakeOrgs{}, &fakeUsers{}, fakeFeatures{})

	if err := service.AddOwner(context.Background(), "org-1", "owner-1"); err != nil {
		t.Fatalf("add owner: %v", err)
	}
	if !store.Has("org-1", "owner-1") {
		t.Fatal("expected owner membership to exist")
	}
}

type fakeOrgs struct {
	owners map[string]string
}

func (s *fakeOrgs) OwnerID(_ context.Context, orgID string) (string, bool, error) {
	owner, ok := s.owners[orgID]
	return owner, ok, nil
}

type fakeUsers struct {
	existing map[string]bool
}

func (s *fakeUsers) Exists(_ context.Context, userID string) (bool, error) {
	return s.existing[userID], nil
}

type fakeFeatures struct{}

func (fakeFeatures) ValidRole(feature, role string) bool {
	switch feature {
	case entities.FeatureOrg, entities.FeatureMember:
		return role == entities.FeatureRoleViewer || role == entities.FeatureRoleAdmin
	case entities.FeatureProject, entities.FeatureExpense:
		return entities.ValidFeatureRole(role)
	default:
		return false
	}
}
