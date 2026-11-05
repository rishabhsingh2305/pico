package projectstest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

func TestCreate(t *testing.T) {
	store := &fakeStore{}
	service := services.NewProjectService(store)

	project, err := service.Create(context.Background(), services.CreateProjectInput{
		OrgID:    "org-1",
		UserID:   "user-1",
		Name:     "  Launch  ",
		Currency: "usd",
		Tags:     []string{" travel ", "food", "food"},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if got, want := project.Name, "Launch"; got != want {
		t.Fatalf("expected name %q, got %q", want, got)
	}
	if got, want := project.Currency, "USD"; got != want {
		t.Fatalf("expected currency %q, got %q", want, got)
	}
	if got, want := project.OrgID, "org-1"; got != want {
		t.Fatalf("expected org %q, got %q", want, got)
	}
	if got, want := project.UserID, "user-1"; got != want {
		t.Fatalf("expected user %q, got %q", want, got)
	}
	if got, want := project.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected tags %v, got %v", want, got)
	}
}

func TestCreateValidation(t *testing.T) {
	service := services.NewProjectService(&fakeStore{})

	for _, input := range []services.CreateProjectInput{
		{Name: "   ", Currency: "USD"},
		{Name: "Launch", Currency: "US"},
		{Name: "Launch", Currency: "USD", Tags: []string{strings.Repeat("a", 33)}},
		{Name: "Launch", Currency: "USD", Tags: make([]string, 21)},
	} {
		_, err := service.Create(context.Background(), input)
		var validationErr *validation.Error
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected validation error, got %v", err)
		}
	}
}

func TestListPaginationAndTags(t *testing.T) {
	store := &fakeStore{}
	service := services.NewProjectService(store)

	if _, err := service.List(context.Background(), services.ListProjectsInput{
		OrgID: "org-1",
		Tags:  []string{" travel ", "food", "food", ""},
	}); err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if got, want := store.lastFilter.Limit, validation.DefaultLimit; got != want {
		t.Fatalf("expected limit %d, got %d", want, got)
	}
	if got, want := store.lastFilter.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected tags %v, got %v", want, got)
	}

	if _, err := service.List(context.Background(), services.ListProjectsInput{OrgID: "org-1", Limit: 101}); err == nil {
		t.Fatal("expected an error for limit > 100")
	}
}

func TestUpdate(t *testing.T) {
	store := &fakeStore{}
	service := services.NewProjectService(store)
	name := "  Renamed "
	tags := []string{" travel ", "food"}

	project, err := service.Update(context.Background(), services.UpdateProjectInput{
		OrgID: "org-1",
		ID:    "project-1",
		Name:  &name,
		Tags:  &tags,
	})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if got, want := project.Name, "Renamed"; got != want {
		t.Fatalf("expected name %q, got %q", want, got)
	}
	if got, want := project.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected tags %v, got %v", want, got)
	}
	if store.lastUpdate.Name == nil || store.lastUpdate.Tags == nil {
		t.Fatalf("expected name and tags in update, got %+v", store.lastUpdate)
	}
}

func TestUpdateRequiresAField(t *testing.T) {
	service := services.NewProjectService(&fakeStore{})

	_, err := service.Update(context.Background(), services.UpdateProjectInput{
		OrgID: "org-1",
		ID:    "project-1",
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	store := &fakeStore{findErr: entities.ErrProjectNotFound}
	service := services.NewProjectService(store)

	_, err := service.Get(context.Background(), "org-1", "missing")
	if !errors.Is(err, entities.ErrProjectNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

type fakeStore struct {
	lastFilter dao.ProjectFilter
	lastUpdate dao.ProjectUpdate
	findErr    error
}

func (s *fakeStore) Insert(_ context.Context, project entities.Project) (entities.Project, error) {
	project.ID = "project-1"
	return project, nil
}

func (s *fakeStore) FindByID(_ context.Context, _, _ string) (entities.Project, error) {
	if s.findErr != nil {
		return entities.Project{}, s.findErr
	}
	return entities.Project{ID: "project-1"}, nil
}

func (s *fakeStore) List(_ context.Context, filter dao.ProjectFilter) ([]entities.Project, error) {
	s.lastFilter = filter
	return []entities.Project{}, nil
}

func (s *fakeStore) Update(_ context.Context, orgID, id string, update dao.ProjectUpdate) (entities.Project, error) {
	s.lastUpdate = update
	project := entities.Project{ID: id, OrgID: orgID, UpdatedAt: update.UpdatedAt}
	if update.Name != nil {
		project.Name = *update.Name
	}
	if update.Tags != nil {
		project.Tags = *update.Tags
	}
	return project, nil
}

func (s *fakeStore) Exists(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
