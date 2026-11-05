package membershipstest

import (
	"context"
	"errors"
	"testing"
	"time"

	"picoservice/internal/entities"
	"picoservice/internal/services"
)

func TestAddMember(t *testing.T) {
	store := newFakeStore()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users)

	membership, err := service.Add(context.Background(), services.CreateMembershipInput{
		ActorID: "owner-1",
		OrgID:   "org-1",
		UserID:  "user-2",
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if got, want := membership.Role, entities.RoleMember; got != want {
		t.Fatalf("expected role %q, got %q", want, got)
	}
	if !store.exists("org-1", "user-2") {
		t.Fatal("expected membership to be stored")
	}
}

func TestAddRejectsNonOwner(t *testing.T) {
	store := newFakeStore()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users)

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		ActorID: "user-2",
		OrgID:   "org-1",
		UserID:  "user-2",
	})
	if !errors.Is(err, entities.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAddRejectsMissingUser(t *testing.T) {
	store := newFakeStore()
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{}
	service := services.NewMembershipService(store, orgs, users)

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		ActorID: "owner-1",
		OrgID:   "org-1",
		UserID:  "missing",
	})
	if !errors.Is(err, entities.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	store := newFakeStore()
	store.add("org-1", "user-2")
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	users := &fakeUsers{existing: map[string]bool{"user-2": true}}
	service := services.NewMembershipService(store, orgs, users)

	_, err := service.Add(context.Background(), services.CreateMembershipInput{
		ActorID: "owner-1",
		OrgID:   "org-1",
		UserID:  "user-2",
	})
	if !errors.Is(err, entities.ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestListDerivesRoles(t *testing.T) {
	store := newFakeStore()
	store.addWithTime("org-1", "owner-1", time.Now().UTC())
	store.addWithTime("org-1", "user-2", time.Now().UTC().Add(-time.Minute))
	orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
	service := services.NewMembershipService(store, orgs, &fakeUsers{})

	items, err := service.List(context.Background(), services.ListMembershipsInput{
		ActorID: "owner-1",
		OrgID:   "org-1",
	})
	if err != nil {
		t.Fatalf("list memberships: %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("expected %d memberships, got %d", want, got)
	}

	roles := map[string]string{}
	for _, item := range items {
		roles[item.UserID] = item.Role
	}
	if got, want := roles["owner-1"], entities.RoleOwner; got != want {
		t.Fatalf("expected owner role %q, got %q", want, got)
	}
	if got, want := roles["user-2"], entities.RoleMember; got != want {
		t.Fatalf("expected member role %q, got %q", want, got)
	}
}

func TestListMissingOrg(t *testing.T) {
	service := services.NewMembershipService(newFakeStore(), &fakeOrgs{}, &fakeUsers{})

	_, err := service.List(context.Background(), services.ListMembershipsInput{OrgID: "missing"})
	if !errors.Is(err, entities.ErrOrgNotFound) {
		t.Fatalf("expected ErrOrgNotFound, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	t.Run("non-owner", func(t *testing.T) {
		store := newFakeStore()
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{})

		err := service.Remove(context.Background(), "user-2", "org-1", "user-3")
		if !errors.Is(err, entities.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("owner cannot remove self", func(t *testing.T) {
		store := newFakeStore()
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{})

		err := service.Remove(context.Background(), "owner-1", "org-1", "owner-1")
		if !errors.Is(err, entities.ErrCannotRemoveOwner) {
			t.Fatalf("expected ErrCannotRemoveOwner, got %v", err)
		}
	})

	t.Run("owner removes member", func(t *testing.T) {
		store := newFakeStore()
		store.add("org-1", "user-2")
		orgs := &fakeOrgs{owners: map[string]string{"org-1": "owner-1"}}
		service := services.NewMembershipService(store, orgs, &fakeUsers{})

		if err := service.Remove(context.Background(), "owner-1", "org-1", "user-2"); err != nil {
			t.Fatalf("remove member: %v", err)
		}
		if store.exists("org-1", "user-2") {
			t.Fatal("expected membership to be removed")
		}
	})
}

func TestAddOwnerAndExists(t *testing.T) {
	store := newFakeStore()
	service := services.NewMembershipService(store, &fakeOrgs{}, &fakeUsers{})

	if err := service.AddOwner(context.Background(), "org-1", "owner-1"); err != nil {
		t.Fatalf("add owner: %v", err)
	}
	exists, err := service.Exists(context.Background(), "org-1", "owner-1")
	if err != nil {
		t.Fatalf("check membership: %v", err)
	}
	if !exists {
		t.Fatal("expected owner membership to exist")
	}
}

type fakeStore struct {
	memberships map[string]entities.Membership
	nextID      int
}

func newFakeStore() *fakeStore {
	return &fakeStore{memberships: make(map[string]entities.Membership)}
}

func (s *fakeStore) add(orgID, userID string) {
	s.addWithTime(orgID, userID, time.Now().UTC())
}

func (s *fakeStore) addWithTime(orgID, userID string, createdAt time.Time) {
	s.nextID++
	id := orgID + "/" + userID
	s.memberships[id] = entities.Membership{
		ID:        id,
		OrgID:     orgID,
		UserID:    userID,
		CreatedAt: createdAt,
	}
}

func (s *fakeStore) exists(orgID, userID string) bool {
	_, ok := s.memberships[orgID+"/"+userID]
	return ok
}

func (s *fakeStore) Insert(_ context.Context, membership entities.Membership) (entities.Membership, error) {
	key := membership.OrgID + "/" + membership.UserID
	if _, ok := s.memberships[key]; ok {
		return entities.Membership{}, entities.ErrAlreadyMember
	}
	s.nextID++
	membership.ID = key
	s.memberships[key] = membership
	return membership, nil
}

func (s *fakeStore) ListByOrg(_ context.Context, orgID string, _, _ int) ([]entities.Membership, error) {
	items := make([]entities.Membership, 0)
	for _, membership := range s.memberships {
		if membership.OrgID == orgID {
			items = append(items, membership)
		}
	}
	return items, nil
}

func (s *fakeStore) Exists(_ context.Context, orgID, userID string) (bool, error) {
	return s.exists(orgID, userID), nil
}

func (s *fakeStore) Delete(_ context.Context, orgID, userID string) error {
	key := orgID + "/" + userID
	if _, ok := s.memberships[key]; !ok {
		return entities.ErrMembershipNotFound
	}
	delete(s.memberships, key)
	return nil
}

func (s *fakeStore) AddOwner(ctx context.Context, orgID, userID string) error {
	_, err := s.Insert(ctx, entities.Membership{
		OrgID:     orgID,
		UserID:    userID,
		Role:      entities.RoleOwner,
		CreatedAt: time.Now().UTC(),
	})
	return err
}

func (s *fakeStore) ListOrgIDsByUser(_ context.Context, userID string, _, _ int) ([]string, error) {
	ids := make([]string, 0)
	for _, membership := range s.memberships {
		if membership.UserID == userID {
			ids = append(ids, membership.OrgID)
		}
	}
	return ids, nil
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
