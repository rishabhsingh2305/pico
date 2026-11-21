package fakes

import (
	"context"

	"picoservice/internal/services"
)

// LinkAnalyticsService is a configurable fake link stats service.
type LinkAnalyticsService struct {
	StatsFunc func(context.Context, services.LinkStatsInput) (services.LinkStats, error)
}

func (s *LinkAnalyticsService) Stats(ctx context.Context, in services.LinkStatsInput) (services.LinkStats, error) {
	if s.StatsFunc != nil {
		return s.StatsFunc(ctx, in)
	}
	return services.LinkStats{Items: []services.LinkStat{}}, nil
}
