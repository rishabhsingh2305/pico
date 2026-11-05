package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

// OrgMembershipStore provides the membership operations organization creation
// and listing need without depending on the full membership DAO.
type OrgMembershipStore interface {
	AddOwner(ctx context.Context, orgID, userID string) error
	ListOrgIDsByUser(ctx context.Context, userID string, limit, offset int) ([]string, error)
}

// CreateOrgInput contains the fields needed to create an organization.
type CreateOrgInput struct {
	ActorID string
	Name    string
	Tags    []string
}

// UpdateOrgInput contains optional organization fields to update.
type UpdateOrgInput struct {
	ActorID string
	OrgID   string
	Name    *string
	Tags    *[]string
}

// ListOrgsInput contains pagination and tag filters for listing an actor's
// organizations.
type ListOrgsInput struct {
	ActorID string
	Tags    []string
	Limit   int
	Offset  int
}

// OrgService applies organization validation and business rules.
type OrgService struct {
	store       dao.OrgDAO
	memberships OrgMembershipStore
}

// NewOrgService returns an organization service backed by store and
// memberships.
func NewOrgService(store dao.OrgDAO, memberships OrgMembershipStore) *OrgService {
	return &OrgService{store: store, memberships: memberships}
}

// Create validates and stores an organization, then adds its owner membership.
func (s *OrgService) Create(ctx context.Context, in CreateOrgInput) (entities.Org, error) {
	name, err := validation.Name(in.Name)
	if err != nil {
		return entities.Org{}, err
	}
	tags, err := normalizeTags(in.Tags)
	if err != nil {
		return entities.Org{}, err
	}

	now := time.Now().UTC()
	org := entities.Org{
		Name:      name,
		UserID:    in.ActorID,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	created, err := s.store.Insert(ctx, org)
	if err != nil {
		return entities.Org{}, fmt.Errorf("create org: %w", err)
	}

	if err := s.memberships.AddOwner(ctx, created.ID, in.ActorID); err != nil {
		// Use a fresh context so cleanup still has a chance when the request
		// context has been canceled or its deadline has expired.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cleanupCancel()

		cleanupErr := s.store.Delete(cleanupCtx, created.ID)
		if cleanupErr != nil {
			return entities.Org{}, fmt.Errorf("create owner membership: %w", errors.Join(err, fmt.Errorf("cleanup org: %w", cleanupErr)))
		}
		return entities.Org{}, fmt.Errorf("create owner membership: %w", err)
	}

	return created, nil
}

// Get returns an organization by ID.
func (s *OrgService) Get(ctx context.Context, id string) (entities.Org, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	org, err := s.store.FindByID(ctx, id)
	if err != nil {
		return entities.Org{}, fmt.Errorf("get org: %w", err)
	}
	return org, nil
}

// List returns organizations the actor belongs to.
func (s *OrgService) List(ctx context.Context, in ListOrgsInput) ([]entities.Org, error) {
	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	orgIDs, err := s.memberships.ListOrgIDsByUser(ctx, in.ActorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	if len(orgIDs) == 0 {
		return []entities.Org{}, nil
	}

	items, err := s.store.FindByIDs(ctx, orgIDs, normalizeTagFilter(in.Tags))
	if err != nil {
		return nil, fmt.Errorf("list orgs: %w", err)
	}

	byID := make(map[string]entities.Org, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	ordered := make([]entities.Org, 0, len(orgIDs))
	for _, id := range orgIDs {
		if org, ok := byID[id]; ok {
			ordered = append(ordered, org)
		}
	}
	return ordered, nil
}

// Update applies optional name and tag changes to an organization. Only its
// owner may update it.
func (s *OrgService) Update(ctx context.Context, in UpdateOrgInput) (entities.Org, error) {
	if in.Name == nil && in.Tags == nil {
		return entities.Org{}, &validation.Error{Field: "body", Message: "at least one field is required"}
	}

	update := dao.OrgUpdate{UpdatedAt: time.Now().UTC()}
	if in.Name != nil {
		name, err := validation.Name(*in.Name)
		if err != nil {
			return entities.Org{}, err
		}
		update.Name = &name
	}
	if in.Tags != nil {
		tags, err := normalizeTags(*in.Tags)
		if err != nil {
			return entities.Org{}, err
		}
		update.Tags = &tags
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	current, err := s.store.FindByID(ctx, in.OrgID)
	if err != nil {
		return entities.Org{}, fmt.Errorf("get org: %w", err)
	}
	if current.UserID != in.ActorID {
		return entities.Org{}, entities.ErrForbidden
	}

	updated, err := s.store.Update(ctx, in.OrgID, update)
	if err != nil {
		return entities.Org{}, fmt.Errorf("update org: %w", err)
	}
	return updated, nil
}

// Exists reports whether an organization exists. It is used by identity
// middleware.
func (s *OrgService) Exists(ctx context.Context, id string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	exists, err := s.store.Exists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check org: %w", err)
	}
	return exists, nil
}
