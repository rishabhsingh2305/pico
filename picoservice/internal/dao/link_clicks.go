package dao

import (
	"context"
	"time"
)

// LinkClickCount is an hourly click total for one link.
type LinkClickCount struct {
	OrgID     string
	LinkID    string
	HourStart time.Time
	Count     int64
}

// LinkClickTotal is a click total for one link over a requested range.
type LinkClickTotal struct {
	LinkID string
	Clicks int64
}

// LinkStatsFilter contains the link analytics read filters.
type LinkStatsFilter struct {
	OrgID   string
	LinkIDs []string
	From    time.Time
	To      time.Time
}

// LinkClickWriter persists hourly click totals.
type LinkClickWriter interface {
	UpsertHourly(ctx context.Context, counts []LinkClickCount) error
}

// LinkClickReader reads aggregated click totals.
type LinkClickReader interface {
	SumByLinks(ctx context.Context, filter LinkStatsFilter) ([]LinkClickTotal, error)
}

// LinkAnalyticsState is the ingestion watermark and lease for one pull zone.
type LinkAnalyticsState struct {
	PullZoneID        int64
	LastCompletedHour time.Time
	LockedUntil       *time.Time
	UpdatedAt         time.Time
}

// LinkAnalyticsStateStore manages the job watermark and lease.
type LinkAnalyticsStateStore interface {
	Acquire(ctx context.Context, pullZoneID int64, now, leaseUntil, initialHour time.Time) (LinkAnalyticsState, bool, error)
	Advance(ctx context.Context, pullZoneID int64, hour time.Time) error
	Release(ctx context.Context, pullZoneID int64, now time.Time) error
}
