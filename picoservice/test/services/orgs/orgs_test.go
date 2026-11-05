package orgstest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

func TestCreateAddsOwnerMembership(t *testing.T) {
	store := newFakeStore()
	memberships := &fakeMembershipStore{}
	service := services.NewOrgService(store, memberships)

	org, err := service.Create(context.Background(), services.CreateOrgInput{
		ActorID: "user-1",
		Name:    "  Acme  ",
		Tags:    []string{" team ", "team", "finance"},
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	if got, want := org.Name, "Acme"; got != want {
		t.Fatalf("expected name %q, got %q", want, got)
	}
	if got, want := org.UserID, "user-1"; got != want {
		t.Fatalf("expected owner %q, got %q", want, got)
	}
	if got, want := len(org.Tags), 2; got != want {
		t.Fatalf("expected %d tags, got %d", want, got)
	}
	if got, want := org.Tags[0], "finance"; got != want {
		t.Fatalf("expected first tag %q, got %q", want, got)
	}
	if got, want := org.Tags[1], "team"; got != want {
		t.Fatalf("expected second tag %q, got %q", want, got)
	}
	if got, want := memberships.added[org.ID], "user-1"; got != want {
		t.Fatalf("expected owner membership for %q, got %q", want, got)
	}
}

func TestCreateCleansUpWhenOwnerMembershipFails(t *testing.T) {
	store := newFakeStore()
	memberships := &fakeMembershipStore{addErr: errors.New("insert membership")}
	service := services.NewOrgService(store, memberships)

	_, err := service.Create(context.Background(), services.CreateOrgInput{
		ActorID: "user-1",
		Name:    "Acme",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if len(store.orgs) != 0 {
		t.Fatalf("expected the org to be cleaned up, got %d orgs", len(store.orgs))
	}
}

func TestListPreservesMembershipOrder(t *testing.T) {
	store := newFakeStore()
	store.addOrg("org-1", "One", "user-1")
	store.addOrg("org-2", "Two", "user-1")
	memberships := &fakeMembershipStore{orgIDs: []string{"org-2", "org-1"}}
	service := services.NewOrgService(store, memberships)

	items, err := service.List(context.Background(), services.ListOrgsInput{ActorID: "user-1"})
	if err != nil {
		t.Fatalf("list orgs: %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("expected %d orgs, got %d", want, got)
	}
	if got, want := items[0].ID, "org-2"; got != want {
		t.Fatalf("expected first org %q, got %q", want, got)
	}
	if got, want := items[1].ID, "org-1"; got != want {
		t.Fatalf("expected second org %q, got %q", want, got)
	}
}

func TestListUsesPagination(t *testing.T) {
	store := newFakeStore()
	memberships := &fakeMembershipStore{}
	service := services.NewOrgService(store, memberships)

	if _, err := service.List(context.Background(), services.ListOrgsInput{ActorID: "user-1"}); err != nil {
		t.Fatalf("list orgs: %v", err)
	}
	if got, want := memberships.lastLimit, validation.DefaultLimit; got != want {
		t.Fatalf("expected limit %d, got %d", want, got)
	}
}

func TestListFiltersByTags(t *testing.T) {
	store := newFakeStore()
	store.addOrgWithTags("org-1", "One", "user-1", []string{"finance", "team"})
	store.addOrgWithTags("org-2", "Two", "user-1", []string{"travel"})
	memberships := &fakeMembershipStore{orgIDs: []string{"org-1", "org-2"}}
	service := services.NewOrgService(store, memberships)

	items, err := service.List(context.Background(), services.ListOrgsInput{
		ActorID: "user-1",
		Tags:    []string{"team", "other"},
	})
	if err != nil {
		t.Fatalf("list orgs: %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("expected %d org, got %d", want, got)
	}
	if got, want := items[0].ID, "org-1"; got != want {
		t.Fatalf("expected org %q, got %q", want, got)
	}
}

func TestUpdateRequiresOwner(t *testing.T) {
	store := newFakeStore()
	store.addOrg("org-1", "Acme", "user-1")
	service := services.NewOrgService(store, &fakeMembershipStore{})
	name := "New name"

	_, err := service.Update(context.Background(), services.UpdateOrgInput{
		ActorID: "user-2",
		OrgID:   "org-1",
		Name:    &name,
	})
	if !errors.Is(err, entities.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateRenamesForOwner(t *testing.T) {
	store := newFakeStore()
	store.addOrg("org-1", "Acme", "user-1")
	service := services.NewOrgService(store, &fakeMembershipStore{})
	name := "  New name "

	org, err := service.Update(context.Background(), services.UpdateOrgInput{
		ActorID: "user-1",
		OrgID:   "org-1",
		Name:    &name,
	})
	if err != nil {
		t.Fatalf("update org: %v", err)
	}
	if got, want := org.Name, "New name"; got != want {
		t.Fatalf("expected name %q, got %q", want, got)
	}
}

func TestUpdateTagsOnly(t *testing.T) {
	store := newFakeStore()
	store.addOrg("org-1", "Acme", "user-1")
	service := services.NewOrgService(store, &fakeMembershipStore{})
	tags := []string{" team ", "team", "finance"}

	org, err := service.Update(context.Background(), services.UpdateOrgInput{
		ActorID: "user-1",
		OrgID:   "org-1",
		Tags:    &tags,
	})
	if err != nil {
		t.Fatalf("update org tags: %v", err)
	}
	if got, want := org.Name, "Acme"; got != want {
		t.Fatalf("expected name to remain %q, got %q", want, got)
	}
	if got, want := len(org.Tags), 2; got != want {
		t.Fatalf("expected %d tags, got %d", want, got)
	}
	if got, want := org.Tags[0], "finance"; got != want {
		t.Fatalf("expected first tag %q, got %q", want, got)
	}
	if got, want := org.Tags[1], "team"; got != want {
		t.Fatalf("expected second tag %q, got %q", want, got)
	}
}

func TestUpdateRequiresAtLeastOneField(t *testing.T) {
	store := newFakeStore()
	store.addOrg("org-1", "Acme", "user-1")
	service := services.NewOrgService(store, &fakeMembershipStore{})

	_, err := service.Update(context.Background(), services.UpdateOrgInput{
		ActorID: "user-1",
		OrgID:   "org-1",
	})
	if err == nil {
		t.Fatal("expected a validation error")
	}
}

func TestGetAndExists(t *testing.T) {
	store := newFakeStore()
	store.addOrg("org-1", "Acme", "user-1")
	service := services.NewOrgService(store, &fakeMembershipStore{})

	if _, err := service.Get(context.Background(), "org-1"); err != nil {
		t.Fatalf("get org: %v", err)
	}
	exists, err := service.Exists(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("check org: %v", err)
	}
	if !exists {
		t.Fatal("expected org to exist")
	}
}

type fakeStore struct {
	orgs      map[string]entities.Org
	nextID    int
	insertErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{orgs: make(map[string]entities.Org)}
}

func (s *fakeStore) addOrg(id, name, ownerID string) entities.Org {
	now := time.Now().UTC()
	org := entities.Org{ID: id, Name: name, UserID: ownerID, CreatedAt: now, UpdatedAt: now}
	s.orgs[id] = org
	return org
}

func (s *fakeStore) addOrgWithTags(id, name, ownerID string, tags []string) entities.Org {
	org := s.addOrg(id, name, ownerID)
	org.Tags = tags
	s.orgs[id] = org
	return org
}

func (s *fakeStore) Insert(_ context.Context, org entities.Org) (entities.Org, error) {
	if s.insertErr != nil {
		return entities.Org{}, s.insertErr
	}
	s.nextID++
	org.ID = fmt.Sprintf("org-%d", s.nextID)
	s.orgs[org.ID] = org
	return org, nil
}

func (s *fakeStore) FindByID(_ context.Context, id string) (entities.Org, error) {
	org, ok := s.orgs[id]
	if !ok {
		return entities.Org{}, entities.ErrOrgNotFound
	}
	return org, nil
}

func (s *fakeStore) FindByIDs(_ context.Context, ids []string, tags []string) ([]entities.Org, error) {
	items := make([]entities.Org, 0, len(ids))
	for _, id := range ids {
		org, ok := s.orgs[id]
		if !ok {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(org.Tags, tags) {
			continue
		}
		items = append(items, org)
	}
	return items, nil
}

func (s *fakeStore) Update(_ context.Context, id string, update dao.OrgUpdate) (entities.Org, error) {
	org, ok := s.orgs[id]
	if !ok {
		return entities.Org{}, entities.ErrOrgNotFound
	}
	if update.Name != nil {
		org.Name = *update.Name
	}
	if update.Tags != nil {
		org.Tags = *update.Tags
	}
	org.UpdatedAt = update.UpdatedAt
	s.orgs[id] = org
	return org, nil
}

func (s *fakeStore) Delete(_ context.Context, id string) error {
	delete(s.orgs, id)
	return nil
}

func (s *fakeStore) Exists(_ context.Context, id string) (bool, error) {
	_, ok := s.orgs[id]
	return ok, nil
}

func (s *fakeStore) OwnerID(_ context.Context, id string) (string, bool, error) {
	org, ok := s.orgs[id]
	if !ok {
		return "", false, nil
	}
	return org.UserID, true, nil
}

type fakeMembershipStore struct {
	added      map[string]string
	addErr     error
	orgIDs     []string
	lastLimit  int
	lastOffset int
}

func (s *fakeMembershipStore) AddOwner(_ context.Context, orgID, userID string) error {
	if s.addErr != nil {
		return s.addErr
	}
	if s.added == nil {
		s.added = make(map[string]string)
	}
	s.added[orgID] = userID
	return nil
}

func (s *fakeMembershipStore) ListOrgIDsByUser(_ context.Context, _ string, limit, offset int) ([]string, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	return s.orgIDs, nil
}

func hasAnyTag(tags, filter []string) bool {
	for _, tag := range tags {
		for _, wanted := range filter {
			if tag == wanted {
				return true
			}
		}
	}
	return false
}
