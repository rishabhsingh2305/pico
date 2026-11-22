package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// AnalyticsSourceFilter contains analytics source list filters.
type AnalyticsSourceFilter struct {
	OrgID      string
	Query      string
	ActiveOnly bool
	Limit      int
	Offset     int
}

// AnalyticsSourceUpdate contains optional analytics source fields.
type AnalyticsSourceUpdate struct {
	Name        *string
	Description *string
	Events      *[]string
	Disabled    *bool
	UpdatedAt   time.Time
}

// AnalyticsSourceDAO persists analytics source records.
type AnalyticsSourceDAO interface {
	Insert(ctx context.Context, source entities.AnalyticsSource) (entities.AnalyticsSource, error)
	FindByID(ctx context.Context, orgID, id string) (entities.AnalyticsSource, error)
	FindByKey(ctx context.Context, key string) (entities.AnalyticsSource, error)
	FindByKeys(ctx context.Context, keys []string) ([]entities.AnalyticsSource, error)
	List(ctx context.Context, filter AnalyticsSourceFilter) ([]entities.AnalyticsSource, error)
	Update(ctx context.Context, orgID, id string, update AnalyticsSourceUpdate) (entities.AnalyticsSource, error)
	Delete(ctx context.Context, orgID, id string) error
}

// AnalyticsCaptureCount is an hourly capture counter at a dimension grain.
type AnalyticsCaptureCount struct {
	OrgID        string
	SourceID     string
	Event        string
	HourStart    time.Time
	CountryCode  string
	Device       string
	IsBot        bool
	ReferrerHost string
	Count        int64
}

// AnalyticsCaptureFilter contains capture stats filters.
type AnalyticsCaptureFilter struct {
	OrgID     string
	SourceID  string
	From      time.Time
	To        time.Time
	Interval  string
	Event     string
	Countries []string
	Devices   []string
	Bot       *bool
	Referrer  string
}

// AnalyticsBulkFilter contains capture totals for several sources.
type AnalyticsBulkFilter struct {
	OrgID     string
	SourceIDs []string
	From      time.Time
	To        time.Time
}

// AnalyticsSourceTotal is the capture total for one source.
type AnalyticsSourceTotal struct {
	SourceID string
	Count    int64
}

// AnalyticsBreakdown is one dimension value and its count.
type AnalyticsBreakdown struct {
	Value string
	Count int64
}

// AnalyticsEventTotal is an event and its count.
type AnalyticsEventTotal struct {
	Event string
	Count int64
}

// AnalyticsSeriesPoint is one time bucket and its count.
type AnalyticsSeriesPoint struct {
	Bucket time.Time
	Count  int64
}

// AnalyticsStats is the full dashboard response for one source.
type AnalyticsStats struct {
	Total     int64
	Series    []AnalyticsSeriesPoint
	Events    []AnalyticsEventTotal
	Countries []AnalyticsBreakdown
	Devices   []AnalyticsBreakdown
	Bots      []AnalyticsBreakdown
	Referrers []AnalyticsBreakdown
}

// AnalyticsCaptureWriter persists capture counters.
type AnalyticsCaptureWriter interface {
	UpsertHourly(ctx context.Context, counts []AnalyticsCaptureCount) error
}

// AnalyticsCaptureReader reads capture totals and breakdowns.
type AnalyticsCaptureReader interface {
	Stats(ctx context.Context, filter AnalyticsCaptureFilter) (AnalyticsStats, error)
	SumBySources(ctx context.Context, filter AnalyticsBulkFilter) ([]AnalyticsSourceTotal, error)
}
