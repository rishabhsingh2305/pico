package services

import (
	"context"
	"fmt"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

// CreateProjectInput contains the fields needed to create a project.
type CreateProjectInput struct {
	OrgID    string
	UserID   string
	Name     string
	Currency string
	Tags     []string
}

// UpdateProjectInput contains optional project fields to update.
type UpdateProjectInput struct {
	OrgID string
	ID    string
	Name  *string
	Tags  *[]string
}

// ListProjectsInput contains the project list filters.
type ListProjectsInput struct {
	OrgID  string
	Tags   []string
	Limit  int
	Offset int
}

// ProjectService applies project validation and business rules.
type ProjectService struct {
	store dao.ProjectDAO
}

// NewProjectService returns a project service backed by store.
func NewProjectService(store dao.ProjectDAO) *ProjectService {
	return &ProjectService{store: store}
}

// Create validates and stores a project.
func (s *ProjectService) Create(ctx context.Context, in CreateProjectInput) (entities.Project, error) {
	name, err := validation.Name(in.Name)
	if err != nil {
		return entities.Project{}, err
	}
	currency, err := validation.Currency(in.Currency)
	if err != nil {
		return entities.Project{}, err
	}
	tags, err := normalizeTags(in.Tags)
	if err != nil {
		return entities.Project{}, err
	}

	now := time.Now().UTC()
	project := entities.Project{
		OrgID:     in.OrgID,
		UserID:    in.UserID,
		Name:      name,
		Currency:  currency,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	created, err := s.store.Insert(ctx, project)
	if err != nil {
		return entities.Project{}, fmt.Errorf("create project: %w", err)
	}
	return created, nil
}

// Get returns a project scoped to orgID.
func (s *ProjectService) Get(ctx context.Context, orgID, id string) (entities.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	project, err := s.store.FindByID(ctx, orgID, id)
	if err != nil {
		return entities.Project{}, fmt.Errorf("get project: %w", err)
	}
	return project, nil
}

// List returns projects scoped to orgID. Tags use any-match semantics.
func (s *ProjectService) List(ctx context.Context, in ListProjectsInput) ([]entities.Project, error) {
	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	items, err := s.store.List(ctx, dao.ProjectFilter{
		OrgID:  in.OrgID,
		Tags:   normalizeTagFilter(in.Tags),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return items, nil
}

// Update applies optional name and tag changes to a project.
func (s *ProjectService) Update(ctx context.Context, in UpdateProjectInput) (entities.Project, error) {
	if in.Name == nil && in.Tags == nil {
		return entities.Project{}, &validation.Error{Field: "body", Message: "at least one field is required"}
	}

	update := dao.ProjectUpdate{UpdatedAt: time.Now().UTC()}
	if in.Name != nil {
		name, err := validation.Name(*in.Name)
		if err != nil {
			return entities.Project{}, err
		}
		update.Name = &name
	}
	if in.Tags != nil {
		tags, err := normalizeTags(*in.Tags)
		if err != nil {
			return entities.Project{}, err
		}
		update.Tags = &tags
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	updated, err := s.store.Update(ctx, in.OrgID, in.ID, update)
	if err != nil {
		return entities.Project{}, fmt.Errorf("update project: %w", err)
	}
	return updated, nil
}
