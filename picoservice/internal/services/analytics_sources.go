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

// CreateAnalyticsSourceInput contains fields for a new analytics source.
type CreateAnalyticsSourceInput struct {
	OrgID       string
	Name        string
	Description string
	Events      []string
}

// UpdateAnalyticsSourceInput contains optional analytics source fields.
type UpdateAnalyticsSourceInput struct {
	OrgID       string
	ID          string
	Name        *string
	Description *string
	Events      *[]string
	Disabled    *bool
}

// ListAnalyticsSourcesInput contains analytics source list filters.
type ListAnalyticsSourcesInput struct {
	OrgID      string
	Query      string
	ActiveOnly bool
	Limit      int
	Offset     int
}

// AnalyticsSourceList is one page of analytics sources.
type AnalyticsSourceList struct {
	Items      []entities.AnalyticsSource
	NextOffset *int
}

// AnalyticsSourceService applies analytics source business rules.
type AnalyticsSourceService struct {
	store dao.AnalyticsSourceDAO
}

// NewAnalyticsSourceService returns a source service backed by store.
func NewAnalyticsSourceService(store dao.AnalyticsSourceDAO) *AnalyticsSourceService {
	return &AnalyticsSourceService{store: store}
}

// Create validates and stores an analytics source.
func (s *AnalyticsSourceService) Create(ctx context.Context, in CreateAnalyticsSourceInput) (entities.AnalyticsSource, error) {
	if strings.TrimSpace(in.OrgID) == "" {
		return entities.AnalyticsSource{}, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}
	name, err := validation.Name(in.Name)
	if err != nil {
		return entities.AnalyticsSource{}, err
	}
	description, err := validateAnalyticsDescription(in.Description)
	if err != nil {
		return entities.AnalyticsSource{}, err
	}
	eventsInput := in.Events
	if len(eventsInput) == 0 {
		eventsInput = []string{"view"}
	}
	events, err := normalizeAnalyticsEvents(eventsInput)
	if err != nil {
		return entities.AnalyticsSource{}, err
	}
	now := time.Now().UTC()
	source := entities.AnalyticsSource{
		OrgID:       in.OrgID,
		Name:        name,
		Description: description,
		Events:      events,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		key, err := randomAnalyticsKey()
		if err != nil {
			return entities.AnalyticsSource{}, fmt.Errorf("create analytics source: generate key: %w", err)
		}
		source.Key = key
		created, err := s.store.Insert(ctx, source)
		if err == nil {
			return created, nil
		}
		if !errors.Is(err, entities.ErrAnalyticsKeyTaken) {
			return entities.AnalyticsSource{}, fmt.Errorf("create analytics source: %w", err)
		}
	}
	return entities.AnalyticsSource{}, fmt.Errorf("create analytics source: could not generate a unique key")
}

// List returns one page of analytics sources.
func (s *AnalyticsSourceService) List(ctx context.Context, in ListAnalyticsSourcesInput) (AnalyticsSourceList, error) {
	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return AnalyticsSourceList{}, err
	}
	query := strings.ToLower(truncateAnalyticsQuery(in.Query))
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	items, err := s.store.List(ctx, dao.AnalyticsSourceFilter{
		OrgID:      in.OrgID,
		Query:      query,
		ActiveOnly: in.ActiveOnly,
		Limit:      limit + 1,
		Offset:     offset,
	})
	if err != nil {
		return AnalyticsSourceList{}, fmt.Errorf("list analytics sources: %w", err)
	}
	var nextOffset *int
	if len(items) > limit {
		items = items[:limit]
		next := offset + limit
		nextOffset = &next
	}
	return AnalyticsSourceList{Items: items, NextOffset: nextOffset}, nil
}

// Get returns one analytics source scoped to orgID.
func (s *AnalyticsSourceService) Get(ctx context.Context, orgID, id string) (entities.AnalyticsSource, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	source, err := s.store.FindByID(ctx, orgID, id)
	if err != nil {
		return entities.AnalyticsSource{}, fmt.Errorf("get analytics source: %w", err)
	}
	return source, nil
}

// Update applies partial changes to an analytics source.
func (s *AnalyticsSourceService) Update(ctx context.Context, in UpdateAnalyticsSourceInput) (entities.AnalyticsSource, error) {
	if in.Name == nil && in.Description == nil && in.Events == nil && in.Disabled == nil {
		return entities.AnalyticsSource{}, &validation.Error{Field: "body", Message: "at least one field is required"}
	}
	update := dao.AnalyticsSourceUpdate{UpdatedAt: time.Now().UTC()}
	if in.Name != nil {
		name, err := validation.Name(*in.Name)
		if err != nil {
			return entities.AnalyticsSource{}, err
		}
		update.Name = &name
	}
	if in.Description != nil {
		description, err := validateAnalyticsDescription(*in.Description)
		if err != nil {
			return entities.AnalyticsSource{}, err
		}
		update.Description = &description
	}
	if in.Events != nil {
		events, err := normalizeAnalyticsEvents(*in.Events)
		if err != nil {
			return entities.AnalyticsSource{}, err
		}
		update.Events = &events
	}
	update.Disabled = in.Disabled

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	updated, err := s.store.Update(ctx, in.OrgID, in.ID, update)
	if err != nil {
		return entities.AnalyticsSource{}, fmt.Errorf("update analytics source: %w", err)
	}
	return updated, nil
}

// Delete removes an analytics source. Historical counters remain.
func (s *AnalyticsSourceService) Delete(ctx context.Context, orgID, id string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := s.store.Delete(ctx, orgID, id); err != nil {
		return fmt.Errorf("delete analytics source: %w", err)
	}
	return nil
}
