package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

// CreateLinkInput contains the fields needed to create a short link.
type CreateLinkInput struct {
	OrgID  string
	UserID string
	URL    string
	Slug   string
	Title  string
	Tags   []string
}

// UpdateLinkInput contains optional link fields. A nil field is left unchanged.
// The slug is immutable.
type UpdateLinkInput struct {
	OrgID    string
	ID       string
	URL      *string
	Title    *string
	Tags     *[]string
	Disabled *bool
}

// ListLinksInput contains the link list filters.
type ListLinksInput struct {
	OrgID  string
	Query  string
	Tags   []string
	Limit  int
	Offset int
}

// LinkList contains one page of links.
type LinkList struct {
	Items      []entities.Link
	NextOffset *int
}

// LinkService applies link validation and business rules.
type LinkService struct {
	store dao.LinkDAO
}

// NewLinkService returns a link service backed by store.
func NewLinkService(store dao.LinkDAO) *LinkService {
	return &LinkService{store: store}
}

// Create validates and stores a link. An empty slug generates a random alias.
func (s *LinkService) Create(ctx context.Context, in CreateLinkInput) (entities.Link, error) {
	target, err := validateLinkURL(in.URL)
	if err != nil {
		return entities.Link{}, err
	}
	title, err := validateLinkTitle(in.Title)
	if err != nil {
		return entities.Link{}, err
	}
	tags, err := normalizeTags(in.Tags)
	if err != nil {
		return entities.Link{}, err
	}

	now := time.Now().UTC()
	link := entities.Link{
		OrgID:     in.OrgID,
		UserID:    in.UserID,
		URL:       target,
		Title:     title,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	custom := strings.TrimSpace(in.Slug)
	if custom != "" {
		slug, err := validateLinkSlug(custom)
		if err != nil {
			return entities.Link{}, err
		}
		link.Slug = slug
	}

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		if custom == "" {
			slug, err := randomLinkSlug()
			if err != nil {
				return entities.Link{}, fmt.Errorf("create link: generate slug: %w", err)
			}
			link.Slug = slug
		}

		created, err := s.store.Insert(ctx, link)
		if err == nil {
			return created, nil
		}
		if custom != "" || !errors.Is(err, entities.ErrSlugTaken) {
			return entities.Link{}, fmt.Errorf("create link: %w", err)
		}
	}

	return entities.Link{}, fmt.Errorf("create link: could not generate a unique slug")
}

// List returns a page of links scoped to orgID. Query is an alias prefix.
func (s *LinkService) List(ctx context.Context, in ListLinksInput) (LinkList, error) {
	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return LinkList{}, err
	}

	query := strings.ToLower(strings.TrimSpace(in.Query))
	if len(query) > linkQueryMaxLength {
		return LinkList{}, &validation.Error{Field: "q", Message: "q must be at most 64 characters"}
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	items, err := s.store.List(ctx, dao.LinkFilter{
		OrgID:  in.OrgID,
		Query:  query,
		Tags:   normalizeTagFilter(in.Tags),
		Limit:  limit + 1,
		Offset: offset,
	})
	if err != nil {
		return LinkList{}, fmt.Errorf("list links: %w", err)
	}

	var nextOffset *int
	if len(items) > limit {
		items = items[:limit]
		next := offset + limit
		nextOffset = &next
	}
	return LinkList{Items: items, NextOffset: nextOffset}, nil
}

// Get returns a link scoped to orgID.
func (s *LinkService) Get(ctx context.Context, orgID, id string) (entities.Link, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	link, err := s.store.FindByID(ctx, orgID, id)
	if err != nil {
		return entities.Link{}, fmt.Errorf("get link: %w", err)
	}
	return link, nil
}

// Resolve returns the active link for a public redirect.
func (s *LinkService) Resolve(ctx context.Context, slug string) (entities.Link, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return entities.Link{}, entities.ErrLinkNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	link, err := s.store.FindBySlug(ctx, slug)
	if err != nil {
		return entities.Link{}, fmt.Errorf("resolve link: %w", err)
	}
	return link, nil
}

// Update applies partial changes to a link.
func (s *LinkService) Update(ctx context.Context, in UpdateLinkInput) (entities.Link, error) {
	if in.URL == nil && in.Title == nil && in.Tags == nil && in.Disabled == nil {
		return entities.Link{}, &validation.Error{Field: "body", Message: "at least one field is required"}
	}

	update := dao.LinkUpdate{UpdatedAt: time.Now().UTC()}
	if in.URL != nil {
		target, err := validateLinkURL(*in.URL)
		if err != nil {
			return entities.Link{}, err
		}
		update.URL = &target
	}
	if in.Title != nil {
		title, err := validateLinkTitle(*in.Title)
		if err != nil {
			return entities.Link{}, err
		}
		update.Title = &title
	}
	if in.Tags != nil {
		tags, err := normalizeTags(*in.Tags)
		if err != nil {
			return entities.Link{}, err
		}
		update.Tags = &tags
	}
	update.Disabled = in.Disabled

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	updated, err := s.store.Update(ctx, in.OrgID, in.ID, update)
	if err != nil {
		return entities.Link{}, fmt.Errorf("update link: %w", err)
	}
	return updated, nil
}

// Delete tombstones a link. Deleted aliases stay reserved.
func (s *LinkService) Delete(ctx context.Context, orgID, id string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := s.store.Delete(ctx, orgID, id, time.Now().UTC()); err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	return nil
}
