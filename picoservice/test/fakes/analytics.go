package fakes

import (
	"context"

	"picoservice/internal/entities"
	"picoservice/internal/services"
)

// AnalyticsSourceService is a configurable fake analytics source service.
type AnalyticsSourceService struct {
	CreateFunc func(context.Context, services.CreateAnalyticsSourceInput) (entities.AnalyticsSource, error)
	ListFunc   func(context.Context, services.ListAnalyticsSourcesInput) (services.AnalyticsSourceList, error)
	GetFunc    func(context.Context, string, string) (entities.AnalyticsSource, error)
	UpdateFunc func(context.Context, services.UpdateAnalyticsSourceInput) (entities.AnalyticsSource, error)
	DeleteFunc func(context.Context, string, string) error
}

func (s *AnalyticsSourceService) Create(ctx context.Context, in services.CreateAnalyticsSourceInput) (entities.AnalyticsSource, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return entities.AnalyticsSource{ID: "source-1", OrgID: in.OrgID, Name: in.Name, Key: "abc123"}, nil
}

func (s *AnalyticsSourceService) List(ctx context.Context, in services.ListAnalyticsSourcesInput) (services.AnalyticsSourceList, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return services.AnalyticsSourceList{Items: []entities.AnalyticsSource{}}, nil
}

func (s *AnalyticsSourceService) Get(ctx context.Context, orgID, id string) (entities.AnalyticsSource, error) {
	if s.GetFunc != nil {
		return s.GetFunc(ctx, orgID, id)
	}
	return entities.AnalyticsSource{ID: id, OrgID: orgID, Name: "Source", Key: "abc123"}, nil
}

func (s *AnalyticsSourceService) Update(ctx context.Context, in services.UpdateAnalyticsSourceInput) (entities.AnalyticsSource, error) {
	if s.UpdateFunc != nil {
		return s.UpdateFunc(ctx, in)
	}
	return entities.AnalyticsSource{ID: in.ID, OrgID: in.OrgID, Name: "Source", Key: "abc123"}, nil
}

func (s *AnalyticsSourceService) Delete(ctx context.Context, orgID, id string) error {
	if s.DeleteFunc != nil {
		return s.DeleteFunc(ctx, orgID, id)
	}
	return nil
}

// AnalyticsStatsService is a configurable fake analytics stats service.
type AnalyticsStatsService struct {
	StatsFunc     func(context.Context, services.AnalyticsStatsInput) (services.AnalyticsStatsResult, error)
	BulkStatsFunc func(context.Context, services.AnalyticsBulkStatsInput) (services.AnalyticsBulkStatsResult, error)
}

func (s *AnalyticsStatsService) Stats(ctx context.Context, in services.AnalyticsStatsInput) (services.AnalyticsStatsResult, error) {
	if s.StatsFunc != nil {
		return s.StatsFunc(ctx, in)
	}
	return services.AnalyticsStatsResult{}, nil
}

func (s *AnalyticsStatsService) BulkStats(ctx context.Context, in services.AnalyticsBulkStatsInput) (services.AnalyticsBulkStatsResult, error) {
	if s.BulkStatsFunc != nil {
		return s.BulkStatsFunc(ctx, in)
	}
	return services.AnalyticsBulkStatsResult{Items: []services.AnalyticsSourceCaptureTotal{}}, nil
}
