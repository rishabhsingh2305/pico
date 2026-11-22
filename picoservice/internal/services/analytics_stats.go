package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/validation"
)

const (
	// AnalyticsStatsMaxIDs is the largest bulk stats source list.
	AnalyticsStatsMaxIDs = 100
	// AnalyticsStatsMaxRange is the largest accepted capture stats range.
	AnalyticsStatsMaxRange = 365 * 24 * time.Hour
)

// AnalyticsStatsInput contains capture dashboard filters.
type AnalyticsStatsInput struct {
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

// AnalyticsSeriesItem is one capture time bucket.
type AnalyticsSeriesItem struct {
	Bucket time.Time
	Count  int64
}

// AnalyticsEventItem is one event total.
type AnalyticsEventItem struct {
	Event string
	Count int64
}

// AnalyticsDimensionItem is one dimension total.
type AnalyticsDimensionItem struct {
	Value string
	Count int64
}

// AnalyticsStatsResult is the capture dashboard response.
type AnalyticsStatsResult struct {
	Total     int64
	Series    []AnalyticsSeriesItem
	Events    []AnalyticsEventItem
	Countries []AnalyticsDimensionItem
	Devices   []AnalyticsDimensionItem
	Bots      []AnalyticsDimensionItem
	Referrers []AnalyticsDimensionItem
}

// AnalyticsBulkStatsInput contains the list-page capture totals request.
type AnalyticsBulkStatsInput struct {
	OrgID     string
	SourceIDs []string
	From      time.Time
	To        time.Time
}

// AnalyticsSourceCaptureTotal is one source's capture total.
type AnalyticsSourceCaptureTotal struct {
	SourceID string
	Count    int64
}

// AnalyticsBulkStatsResult is the list-page capture totals response.
type AnalyticsBulkStatsResult struct {
	Items []AnalyticsSourceCaptureTotal
}

// AnalyticsStatsReader reads capture dashboards.
type AnalyticsStatsReader struct {
	store dao.AnalyticsCaptureReader
}

// NewAnalyticsStatsReader returns a stats reader backed by store.
func NewAnalyticsStatsReader(store dao.AnalyticsCaptureReader) *AnalyticsStatsReader {
	return &AnalyticsStatsReader{store: store}
}

// Stats returns totals, series, and breakdowns for one source.
func (r *AnalyticsStatsReader) Stats(ctx context.Context, in AnalyticsStatsInput) (AnalyticsStatsResult, error) {
	if r.store == nil {
		return AnalyticsStatsResult{}, errors.New("analytics stats reader: store is required")
	}
	if in.OrgID == "" {
		return AnalyticsStatsResult{}, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}
	if in.SourceID == "" {
		return AnalyticsStatsResult{}, &validation.Error{Field: "sourceID", Message: "sourceID is required"}
	}
	interval, err := analyticsInterval(in.Interval)
	if err != nil {
		return AnalyticsStatsResult{}, err
	}
	if err := validateAnalyticsRange(in.From, in.To); err != nil {
		return AnalyticsStatsResult{}, err
	}
	event := ""
	if in.Event != "" {
		normalized, ok := normalizeAnalyticsEvent(in.Event)
		if !ok {
			return AnalyticsStatsResult{}, &validation.Error{Field: "event", Message: "event is invalid"}
		}
		event = normalized
	}
	countries := normalizeAnalyticsCountries(in.Countries)
	devices := normalizeAnalyticsDevices(in.Devices)
	referrer := truncateAnalyticsQuery(in.Referrer)
	if referrer != "" {
		referrer = strings.ToLower(referrer)
	}

	stats, err := r.store.Stats(ctx, dao.AnalyticsCaptureFilter{
		OrgID:     in.OrgID,
		SourceID:  in.SourceID,
		From:      in.From.UTC(),
		To:        in.To.UTC(),
		Interval:  interval,
		Event:     event,
		Countries: countries,
		Devices:   devices,
		Bot:       in.Bot,
		Referrer:  referrer,
	})
	if err != nil {
		return AnalyticsStatsResult{}, fmt.Errorf("read analytics stats: %w", err)
	}

	result := AnalyticsStatsResult{
		Total:     stats.Total,
		Series:    make([]AnalyticsSeriesItem, 0, len(stats.Series)),
		Events:    make([]AnalyticsEventItem, 0, len(stats.Events)),
		Countries: make([]AnalyticsDimensionItem, 0, len(stats.Countries)),
		Devices:   make([]AnalyticsDimensionItem, 0, len(stats.Devices)),
		Bots:      make([]AnalyticsDimensionItem, 0, len(stats.Bots)),
		Referrers: make([]AnalyticsDimensionItem, 0, len(stats.Referrers)),
	}
	for _, item := range stats.Series {
		result.Series = append(result.Series, AnalyticsSeriesItem{Bucket: item.Bucket, Count: item.Count})
	}
	for _, item := range stats.Events {
		result.Events = append(result.Events, AnalyticsEventItem{Event: item.Event, Count: item.Count})
	}
	for _, item := range stats.Countries {
		result.Countries = append(result.Countries, AnalyticsDimensionItem{Value: item.Value, Count: item.Count})
	}
	for _, item := range stats.Devices {
		result.Devices = append(result.Devices, AnalyticsDimensionItem{Value: item.Value, Count: item.Count})
	}
	for _, item := range stats.Bots {
		result.Bots = append(result.Bots, AnalyticsDimensionItem{Value: item.Value, Count: item.Count})
	}
	for _, item := range stats.Referrers {
		result.Referrers = append(result.Referrers, AnalyticsDimensionItem{Value: item.Value, Count: item.Count})
	}
	return result, nil
}

// BulkStats returns capture totals for several sources.
func (r *AnalyticsStatsReader) BulkStats(ctx context.Context, in AnalyticsBulkStatsInput) (AnalyticsBulkStatsResult, error) {
	if r.store == nil {
		return AnalyticsBulkStatsResult{}, errors.New("analytics stats reader: store is required")
	}
	if in.OrgID == "" {
		return AnalyticsBulkStatsResult{}, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}
	ids := uniqueSortedIDs(in.SourceIDs)
	if len(ids) == 0 {
		return AnalyticsBulkStatsResult{}, &validation.Error{Field: "ids", Message: "ids is required"}
	}
	if len(ids) > AnalyticsStatsMaxIDs {
		return AnalyticsBulkStatsResult{}, &validation.Error{Field: "ids", Message: fmt.Sprintf("ids must contain at most %d items", AnalyticsStatsMaxIDs)}
	}
	if err := validateAnalyticsRange(in.From, in.To); err != nil {
		return AnalyticsBulkStatsResult{}, err
	}
	totals, err := r.store.SumBySources(ctx, dao.AnalyticsBulkFilter{
		OrgID:     in.OrgID,
		SourceIDs: ids,
		From:      in.From.UTC(),
		To:        in.To.UTC(),
	})
	if err != nil {
		return AnalyticsBulkStatsResult{}, fmt.Errorf("read analytics totals: %w", err)
	}
	items := make([]AnalyticsSourceCaptureTotal, 0, len(totals))
	for _, total := range totals {
		items = append(items, AnalyticsSourceCaptureTotal{SourceID: total.SourceID, Count: total.Count})
	}
	return AnalyticsBulkStatsResult{Items: items}, nil
}

func analyticsInterval(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return "day", nil
	case "hour":
		return "hour", nil
	case "day":
		return "day", nil
	default:
		return "", &validation.Error{Field: "interval", Message: "interval must be hour or day"}
	}
}

func validateAnalyticsRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() {
		return &validation.Error{Field: "from", Message: "from and to are required"}
	}
	if !from.Before(to) {
		return &validation.Error{Field: "from", Message: "from must be before to"}
	}
	if to.Sub(from) > AnalyticsStatsMaxRange {
		return &validation.Error{Field: "to", Message: "range must be at most 365 days"}
	}
	return nil
}

func normalizeAnalyticsCountries(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			code := analyticsCountryCode(part)
			if code == "unknown" {
				continue
			}
			if _, ok := seen[code]; ok {
				continue
			}
			seen[code] = struct{}{}
			items = append(items, code)
		}
	}
	sort.Strings(items)
	return items
}

func normalizeAnalyticsDevices(values []string) []string {
	allowed := map[string]struct{}{
		"desktop": {}, "mobile": {}, "tablet": {}, "bot": {}, "unknown": {},
	}
	seen := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			device := strings.ToLower(strings.TrimSpace(part))
			if _, ok := allowed[device]; !ok {
				continue
			}
			if _, ok := seen[device]; ok {
				continue
			}
			seen[device] = struct{}{}
			items = append(items, device)
		}
	}
	sort.Strings(items)
	return items
}
