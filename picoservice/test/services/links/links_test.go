package linkstest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

type fakeLinkStore struct {
	insertFunc     func(context.Context, entities.Link) (entities.Link, error)
	findByIDFunc   func(context.Context, string, string) (entities.Link, error)
	findBySlugFunc func(context.Context, string) (entities.Link, error)
	listFunc       func(context.Context, dao.LinkFilter) ([]entities.Link, error)
	updateFunc     func(context.Context, string, string, dao.LinkUpdate) (entities.Link, error)
	deleteFunc     func(context.Context, string, string, time.Time) (entities.Link, error)
}

func (s fakeLinkStore) Insert(ctx context.Context, link entities.Link) (entities.Link, error) {
	if s.insertFunc != nil {
		return s.insertFunc(ctx, link)
	}
	link.ID = "link-1"
	return link, nil
}

func (s fakeLinkStore) FindByID(ctx context.Context, orgID, id string) (entities.Link, error) {
	if s.findByIDFunc != nil {
		return s.findByIDFunc(ctx, orgID, id)
	}
	return entities.Link{}, entities.ErrLinkNotFound
}

func (s fakeLinkStore) FindBySlug(ctx context.Context, slug string) (entities.Link, error) {
	if s.findBySlugFunc != nil {
		return s.findBySlugFunc(ctx, slug)
	}
	return entities.Link{}, entities.ErrLinkNotFound
}

func (s fakeLinkStore) List(ctx context.Context, filter dao.LinkFilter) ([]entities.Link, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, filter)
	}
	return []entities.Link{}, nil
}

func (s fakeLinkStore) Update(ctx context.Context, orgID, id string, update dao.LinkUpdate) (entities.Link, error) {
	if s.updateFunc != nil {
		return s.updateFunc(ctx, orgID, id, update)
	}
	return entities.Link{}, entities.ErrLinkNotFound
}

func (s fakeLinkStore) Delete(ctx context.Context, orgID, id string, deletedAt time.Time) (entities.Link, error) {
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, orgID, id, deletedAt)
	}
	return entities.Link{}, nil
}

func TestCreateValidatesInput(t *testing.T) {
	store := fakeLinkStore{}
	service := services.NewLinkService(store)

	tests := []struct {
		name    string
		input   services.CreateLinkInput
		field   string
		message string
	}{
		{name: "missing url", input: services.CreateLinkInput{URL: ""}, field: "url", message: "url is required"},
		{name: "unsupported scheme", input: services.CreateLinkInput{URL: "ftp://example.com/file"}, field: "url", message: "url must be a valid http or https URL"},
		{name: "missing host", input: services.CreateLinkInput{URL: "https:///path"}, field: "url", message: "url must be a valid http or https URL"},
		{name: "credentials", input: services.CreateLinkInput{URL: "https://user:pass@example.com"}, field: "url", message: "url must not include credentials"},
		{name: "short slug", input: services.CreateLinkInput{URL: "https://example.com", Slug: "ab"}, field: "slug", message: "slug must be 3 to 32 characters"},
		{name: "invalid slug", input: services.CreateLinkInput{URL: "https://example.com", Slug: "bad slug"}, field: "slug", message: "slug must use lowercase letters, digits, and hyphens"},
		{name: "long title", input: services.CreateLinkInput{URL: "https://example.com", Title: strings.Repeat("a", 121)}, field: "title", message: "title must be at most 120 characters"},
		{name: "too many tags", input: services.CreateLinkInput{URL: "https://example.com", Tags: make([]string, 21)}, field: "tags", message: "tags must contain at most 20 items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.input)
			var validationErr *validation.Error
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected a validation error, got %v", err)
			}
			if validationErr.Field != tt.field || validationErr.Message != tt.message {
				t.Fatalf("expected %s: %q, got %s: %q", tt.field, tt.message, validationErr.Field, validationErr.Message)
			}
		})
	}
}

func TestCreateWithCustomSlug(t *testing.T) {
	var inserted entities.Link
	store := fakeLinkStore{
		insertFunc: func(_ context.Context, link entities.Link) (entities.Link, error) {
			inserted = link
			link.ID = "link-1"
			return link, nil
		},
	}
	service := services.NewLinkService(store)

	created, err := service.Create(context.Background(), services.CreateLinkInput{
		OrgID:  "org-1",
		UserID: "user-1",
		URL:    "HTTPS://example.com/Path?q=1",
		Slug:   " Launch ",
		Title:  " Launch page ",
		Tags:   []string{"Marketing", "launch", "Marketing"},
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	if got, want := inserted.Slug, "launch"; got != want {
		t.Fatalf("expected slug %q, got %q", want, got)
	}
	if got, want := inserted.URL, "https://example.com/Path?q=1"; got != want {
		t.Fatalf("expected url %q, got %q", want, got)
	}
	if got, want := inserted.Title, "Launch page"; got != want {
		t.Fatalf("expected title %q, got %q", want, got)
	}
	if got, want := strings.Join(inserted.Tags, ","), "Marketing,launch"; got != want {
		t.Fatalf("expected tags %q, got %q", want, got)
	}
	if created.ID != "link-1" {
		t.Fatalf("expected created id, got %+v", created)
	}
}

func TestCreateGeneratesSlug(t *testing.T) {
	var inserted entities.Link
	store := fakeLinkStore{
		insertFunc: func(_ context.Context, link entities.Link) (entities.Link, error) {
			inserted = link
			link.ID = "link-1"
			return link, nil
		},
	}
	service := services.NewLinkService(store)

	if _, err := service.Create(context.Background(), services.CreateLinkInput{
		OrgID:  "org-1",
		UserID: "user-1",
		URL:    "https://example.com",
	}); err != nil {
		t.Fatalf("create link: %v", err)
	}

	if got, want := len(inserted.Slug), 7; got != want {
		t.Fatalf("expected slug length %d, got %d (%q)", want, got, inserted.Slug)
	}
	for _, character := range inserted.Slug {
		if !strings.ContainsRune("abcdefghijkmnpqrstuvwxyz23456789", character) {
			t.Fatalf("unexpected slug character %q in %q", character, inserted.Slug)
		}
	}
}

func TestCreateRetriesGeneratedCollisions(t *testing.T) {
	attempts := 0
	store := fakeLinkStore{
		insertFunc: func(_ context.Context, link entities.Link) (entities.Link, error) {
			attempts++
			if attempts < 3 {
				return entities.Link{}, entities.ErrSlugTaken
			}
			link.ID = "link-1"
			return link, nil
		},
	}
	service := services.NewLinkService(store)

	if _, err := service.Create(context.Background(), services.CreateLinkInput{
		OrgID:  "org-1",
		UserID: "user-1",
		URL:    "https://example.com",
	}); err != nil {
		t.Fatalf("create link: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestCreateCustomSlugConflict(t *testing.T) {
	store := fakeLinkStore{
		insertFunc: func(context.Context, entities.Link) (entities.Link, error) {
			return entities.Link{}, entities.ErrSlugTaken
		},
	}
	service := services.NewLinkService(store)

	_, err := service.Create(context.Background(), services.CreateLinkInput{
		OrgID:  "org-1",
		UserID: "user-1",
		URL:    "https://example.com",
		Slug:   "launch",
	})
	if !errors.Is(err, entities.ErrSlugTaken) {
		t.Fatalf("expected ErrSlugTaken, got %v", err)
	}
}

func TestListPaginatesAndFilters(t *testing.T) {
	var got dao.LinkFilter
	store := fakeLinkStore{
		listFunc: func(_ context.Context, filter dao.LinkFilter) ([]entities.Link, error) {
			got = filter
			return []entities.Link{{ID: "1"}, {ID: "2"}, {ID: "3"}}, nil
		},
	}
	service := services.NewLinkService(store)

	result, err := service.List(context.Background(), services.ListLinksInput{
		OrgID:  "org-1",
		Query:  "  Launch ",
		Tags:   []string{"Travel", "Travel"},
		Limit:  2,
		Offset: 4,
	})
	if err != nil {
		t.Fatalf("list links: %v", err)
	}

	if got.Query != "launch" || got.Limit != 3 || got.Offset != 4 || got.OrgID != "org-1" {
		t.Fatalf("unexpected filter: %+v", got)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "Travel" {
		t.Fatalf("expected normalized tags, got %v", got.Tags)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.NextOffset == nil || *result.NextOffset != 6 {
		t.Fatalf("expected next offset 6, got %v", result.NextOffset)
	}
}

func TestListRejectsLongQuery(t *testing.T) {
	service := services.NewLinkService(fakeLinkStore{})

	_, err := service.List(context.Background(), services.ListLinksInput{
		OrgID: "org-1",
		Query: strings.Repeat("a", 65),
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) || validationErr.Field != "q" {
		t.Fatalf("expected a q validation error, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	var got dao.LinkUpdate
	store := fakeLinkStore{
		updateFunc: func(_ context.Context, orgID, id string, update dao.LinkUpdate) (entities.Link, error) {
			if orgID != "org-1" || id != "link-1" {
				t.Fatalf("unexpected scope: %q %q", orgID, id)
			}
			got = update
			return entities.Link{ID: id, OrgID: orgID}, nil
		},
	}
	service := services.NewLinkService(store)

	url := "https://example.com/new"
	title := "New title"
	tags := []string{"launch"}
	disabled := true
	if _, err := service.Update(context.Background(), services.UpdateLinkInput{
		OrgID:    "org-1",
		ID:       "link-1",
		URL:      &url,
		Title:    &title,
		Tags:     &tags,
		Disabled: &disabled,
	}); err != nil {
		t.Fatalf("update link: %v", err)
	}

	if got.URL == nil || *got.URL != url {
		t.Fatalf("expected url %q, got %v", url, got.URL)
	}
	if got.Title == nil || *got.Title != title {
		t.Fatalf("expected title %q, got %v", title, got.Title)
	}
	if got.Tags == nil || len(*got.Tags) != 1 || (*got.Tags)[0] != "launch" {
		t.Fatalf("expected tags, got %v", got.Tags)
	}
	if got.Disabled == nil || !*got.Disabled {
		t.Fatalf("expected disabled update, got %v", got.Disabled)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected updatedAt")
	}
}

func TestUpdateRequiresAField(t *testing.T) {
	service := services.NewLinkService(fakeLinkStore{})

	_, err := service.Update(context.Background(), services.UpdateLinkInput{OrgID: "org-1", ID: "link-1"})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) || validationErr.Field != "body" {
		t.Fatalf("expected a body validation error, got %v", err)
	}
}

func TestResolve(t *testing.T) {
	var gotSlug string
	store := fakeLinkStore{
		findBySlugFunc: func(_ context.Context, slug string) (entities.Link, error) {
			gotSlug = slug
			return entities.Link{Slug: slug, URL: "https://example.com"}, nil
		},
	}
	service := services.NewLinkService(store)

	link, err := service.Resolve(context.Background(), "  Launch ")
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	if gotSlug != "launch" || link.URL != "https://example.com" {
		t.Fatalf("unexpected resolve result: %q %+v", gotSlug, link)
	}
}

func TestDelete(t *testing.T) {
	var gotDeletedAt time.Time
	store := fakeLinkStore{
		deleteFunc: func(_ context.Context, orgID, id string, deletedAt time.Time) (entities.Link, error) {
			if orgID != "org-1" || id != "link-1" {
				t.Fatalf("unexpected scope: %q %q", orgID, id)
			}
			gotDeletedAt = deletedAt
			return entities.Link{ID: id, OrgID: orgID, Slug: "launch"}, nil
		},
	}
	service := services.NewLinkService(store)

	link, err := service.Delete(context.Background(), "org-1", "link-1")
	if err != nil {
		t.Fatalf("delete link: %v", err)
	}
	if gotDeletedAt.IsZero() {
		t.Fatal("expected deletedAt")
	}
	if link.Slug != "launch" {
		t.Fatalf("expected deleted link slug, got %+v", link)
	}
}
