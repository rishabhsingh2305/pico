package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"picoservice/internal/analytics"
	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

const (
	// DefaultLinkAnalyticsPageSize is the Bunny log page size used by the job.
	DefaultLinkAnalyticsPageSize = 5000
	// DefaultLinkAnalyticsLeaseDuration is how long a job run holds its lease.
	DefaultLinkAnalyticsLeaseDuration = 15 * time.Minute
	// DefaultLinkAnalyticsSafetyLag keeps the current partial hour behind
	// recently written logs.
	DefaultLinkAnalyticsSafetyLag = 5 * time.Minute
	// DefaultLinkAnalyticsMaxHoursPerRun bounds catch-up work per invocation.
	DefaultLinkAnalyticsMaxHoursPerRun = 12
	// DefaultLinkAnalyticsInitialBackfill is the first-run lookback.
	DefaultLinkAnalyticsInitialBackfill = 48 * time.Hour
	// DefaultLinkAnalyticsMaxLookback stays inside Bunny's 3-day retention.
	DefaultLinkAnalyticsMaxLookback = 71 * time.Hour
	backfillHourDelay               = 500 * time.Millisecond
)

// LinkSlugResolver resolves link slugs without filtering out disabled or
// deleted records.
type LinkSlugResolver interface {
	FindBySlugs(ctx context.Context, slugs []string) ([]entities.Link, error)
}

// LinkHourProcessor turns one time range of request logs into per-link hourly
// counts. It does not write to the database or manage job state.
type LinkHourProcessor struct {
	logs     analytics.LogSource
	links    LinkSlugResolver
	host     string
	pageSize int
}

// NewLinkHourProcessor returns a processor for one CDN host.
func NewLinkHourProcessor(logs analytics.LogSource, links LinkSlugResolver, host string, pageSize int) *LinkHourProcessor {
	if pageSize <= 0 {
		pageSize = DefaultLinkAnalyticsPageSize
	}
	return &LinkHourProcessor{
		logs:     logs,
		links:    links,
		host:     analytics.NormalizeHost(host),
		pageSize: pageSize,
	}
}

// Process returns counts for one hour, or for the current partial hour when to
// is earlier than the end of the hour.
func (p *LinkHourProcessor) Process(ctx context.Context, from, to time.Time) ([]dao.LinkClickCount, error) {
	if p.logs == nil {
		return nil, errors.New("link hour processor: log source is required")
	}
	if p.links == nil {
		return nil, errors.New("link hour processor: link resolver is required")
	}
	if p.host == "" {
		return nil, errors.New("link hour processor: host is required")
	}
	from = from.UTC()
	to = to.UTC()
	if !to.After(from) {
		return nil, errors.New("link hour processor: to must be after from")
	}

	seen := make(map[string]struct{})
	slugCounts := make(map[string]int64)
	offset := 0
	for {
		page, err := p.logs.List(ctx, analytics.LogQuery{
			From:       from,
			To:         to,
			Host:       p.host,
			PathPrefix: "/s/",
			Status:     "2xx,3xx",
			Limit:      p.pageSize,
			Offset:     offset,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch link logs: %w", err)
		}

		for _, entry := range page.Entries {
			if analytics.NormalizeHost(entry.Host) != p.host {
				continue
			}
			if entry.StatusCode < 200 || entry.StatusCode >= 400 {
				continue
			}

			key := analytics.RequestKey(entry)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			slug, ok := analytics.ParseShortLinkPath(entry.Path)
			if !ok {
				continue
			}
			slugCounts[slug]++
		}

		if !page.HasMore {
			break
		}
		if page.NextOffset <= offset {
			return nil, errors.New("link hour processor: log pagination did not advance")
		}
		offset = page.NextOffset
	}

	if len(slugCounts) == 0 {
		return []dao.LinkClickCount{}, nil
	}

	linksBySlug, err := p.resolveSlugs(ctx, slugCounts)
	if err != nil {
		return nil, err
	}

	byLink := make(map[string]dao.LinkClickCount)
	for slug, count := range slugCounts {
		link, ok := linksBySlug[slug]
		if !ok {
			continue
		}
		current := byLink[link.ID]
		current.OrgID = link.OrgID
		current.LinkID = link.ID
		current.HourStart = from
		current.Count += count
		byLink[link.ID] = current
	}

	items := make([]dao.LinkClickCount, 0, len(byLink))
	for _, count := range byLink {
		items = append(items, count)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LinkID < items[j].LinkID
	})
	return items, nil
}

func (p *LinkHourProcessor) resolveSlugs(ctx context.Context, counts map[string]int64) (map[string]entities.Link, error) {
	slugs := make([]string, 0, len(counts))
	for slug := range counts {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	bySlug := make(map[string]entities.Link, len(slugs))
	for start := 0; start < len(slugs); start += 500 {
		end := start + 500
		if end > len(slugs) {
			end = len(slugs)
		}

		links, err := p.links.FindBySlugs(ctx, slugs[start:end])
		if err != nil {
			return nil, fmt.Errorf("resolve link slugs: %w", err)
		}
		for _, link := range links {
			bySlug[strings.ToLower(link.Slug)] = link
		}
	}
	return bySlug, nil
}

// LinkAnalyticsJobConfig controls the link analytics ingestion job.
type LinkAnalyticsJobConfig struct {
	PullZoneID      int64
	LeaseDuration   time.Duration
	SafetyLag       time.Duration
	MaxHoursPerRun  int
	InitialBackfill time.Duration
	MaxLookback     time.Duration
}

func (c LinkAnalyticsJobConfig) validate() error {
	switch {
	case c.PullZoneID <= 0:
		return errors.New("link analytics: pull zone id is required")
	case c.LeaseDuration <= 0:
		return errors.New("link analytics: lease duration must be positive")
	case c.SafetyLag < 0:
		return errors.New("link analytics: safety lag must not be negative")
	case c.MaxHoursPerRun <= 0:
		return errors.New("link analytics: max hours per run must be positive")
	case c.InitialBackfill <= 0:
		return errors.New("link analytics: initial backfill must be positive")
	case c.MaxLookback <= 0:
		return errors.New("link analytics: max lookback must be positive")
	case c.MaxLookback > 72*time.Hour:
		return errors.New("link analytics: max lookback must not exceed 72 hours")
	default:
		return nil
	}
}

// LinkAnalyticsJobResult summarizes one ingestion run.
type LinkAnalyticsJobResult struct {
	Skipped               bool
	CurrentHourProcessed  bool
	PreviousHourProcessed bool
	HoursProcessed        int
	GapHours              int
	Clicks                int64
	Captures              int64
}

// AnalyticsCapturePipeline groups the optional capture processor and writer.
type AnalyticsCapturePipeline struct {
	Processor *AnalyticsCaptureProcessor
	Writer    dao.AnalyticsCaptureWriter
}

// LinkAnalyticsIngester owns the job lease, watermark, and hour loop.
type LinkAnalyticsIngester struct {
	processor *LinkHourProcessor
	writer    dao.LinkClickWriter
	captures  *AnalyticsCapturePipeline
	state     dao.LinkAnalyticsStateStore
	config    LinkAnalyticsJobConfig
	now       func() time.Time
}

// NewLinkAnalyticsIngester returns an ingester. now is injectable for tests.
func NewLinkAnalyticsIngester(
	processor *LinkHourProcessor,
	writer dao.LinkClickWriter,
	state dao.LinkAnalyticsStateStore,
	config LinkAnalyticsJobConfig,
	now func() time.Time,
) *LinkAnalyticsIngester {
	if now == nil {
		now = time.Now
	}
	return &LinkAnalyticsIngester{
		processor: processor,
		writer:    writer,
		state:     state,
		config:    config,
		now:       now,
	}
}

// SetCapturePipeline enables pixel capture ingestion in the same job run.
func (s *LinkAnalyticsIngester) SetCapturePipeline(pipeline AnalyticsCapturePipeline) {
	if pipeline.Processor == nil || pipeline.Writer == nil {
		return
	}
	s.captures = &pipeline
}

// Run processes the current partial hour, the previous hour for late logs, and
// any missing closed hours.
func (s *LinkAnalyticsIngester) Run(ctx context.Context) (result LinkAnalyticsJobResult, err error) {
	if s.processor == nil || s.writer == nil || s.state == nil {
		return result, errors.New("link analytics job: dependencies are required")
	}
	if s.captures != nil && (s.captures.Processor == nil || s.captures.Writer == nil) {
		return result, errors.New("link analytics job: capture pipeline is incomplete")
	}
	if err := s.config.validate(); err != nil {
		return result, err
	}

	now := s.now().UTC()
	currentHour := now.Truncate(time.Hour)
	initialHour := now.Add(-s.config.InitialBackfill).Truncate(time.Hour)
	state, acquired, err := s.state.Acquire(
		ctx,
		s.config.PullZoneID,
		now,
		now.Add(s.config.LeaseDuration),
		initialHour,
	)
	if err != nil {
		return result, fmt.Errorf("acquire link analytics lease: %w", err)
	}
	if !acquired {
		result.Skipped = true
		return result, nil
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if releaseErr := s.state.Release(releaseCtx, s.config.PullZoneID, time.Now().UTC()); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
	}()

	oldest := now.Add(-s.config.MaxLookback).Truncate(time.Hour)

	if partialTo := now.Add(-s.config.SafetyLag); partialTo.After(currentHour) {
		clicks, captures, err := s.processHour(ctx, currentHour, partialTo)
		if err != nil {
			return result, err
		}
		result.CurrentHourProcessed = true
		result.Clicks += clicks
		result.Captures += captures
	}

	previous := currentHour.Add(-time.Hour)
	if !previous.Before(oldest) {
		clicks, captures, err := s.processHour(ctx, previous, currentHour)
		if err != nil {
			return result, err
		}
		result.PreviousHourProcessed = true
		result.Clicks += clicks
		result.Captures += captures
	}

	last := state.LastCompletedHour.UTC()
	if last.IsZero() {
		last = initialHour
	}
	if last.Before(oldest) {
		result.GapHours = int(oldest.Sub(last) / time.Hour)
		last = oldest
	}
	if last.After(currentHour) {
		last = currentHour
	}

	for processed := 0; last.Before(currentHour) && processed < s.config.MaxHoursPerRun; processed++ {
		hourStart := last
		hourEnd := last.Add(time.Hour)
		clicks, captures, err := s.processHour(ctx, hourStart, hourEnd)
		if err != nil {
			return result, err
		}
		if err := s.state.Advance(ctx, s.config.PullZoneID, hourEnd); err != nil {
			return result, fmt.Errorf("advance link analytics state: %w", err)
		}
		result.HoursProcessed++
		result.Clicks += clicks
		result.Captures += captures
		last = hourEnd
	}

	return result, nil
}

// Backfill recomputes complete hours in [from, to) without changing the
// scheduled job watermark. It is intended for manual recovery after a bug or
// configuration fix.
func (s *LinkAnalyticsIngester) Backfill(ctx context.Context, from, to time.Time) (result LinkAnalyticsJobResult, err error) {
	if s.processor == nil || s.writer == nil || s.state == nil {
		return result, errors.New("link analytics backfill: dependencies are required")
	}
	if s.captures != nil && (s.captures.Processor == nil || s.captures.Writer == nil) {
		return result, errors.New("link analytics backfill: capture pipeline is incomplete")
	}
	if err := s.config.validate(); err != nil {
		return result, err
	}

	now := s.now().UTC()
	currentHour := now.Truncate(time.Hour)
	from = from.UTC().Truncate(time.Hour)
	to = to.UTC().Truncate(time.Hour)
	if to.After(currentHour) {
		to = currentHour
	}
	if !to.After(from) {
		return result, errors.New("link analytics backfill: range must contain at least one complete hour")
	}
	if to.Sub(from) > s.config.MaxLookback {
		return result, fmt.Errorf("link analytics backfill: range must not exceed %s", s.config.MaxLookback)
	}
	if from.Before(now.Add(-s.config.MaxLookback)) {
		return result, fmt.Errorf("link analytics backfill: range starts before the %s lookback window", s.config.MaxLookback)
	}

	_, acquired, err := s.state.Acquire(
		ctx,
		s.config.PullZoneID,
		now,
		now.Add(s.config.LeaseDuration),
		from,
	)
	if err != nil {
		return result, fmt.Errorf("acquire link analytics lease: %w", err)
	}
	if !acquired {
		result.Skipped = true
		return result, nil
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if releaseErr := s.state.Release(releaseCtx, s.config.PullZoneID, time.Now().UTC()); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
	}()

	for hour := from; hour.Before(to); hour = hour.Add(time.Hour) {
		clicks, captures, err := s.processHour(ctx, hour, hour.Add(time.Hour))
		if err != nil {
			return result, err
		}
		result.HoursProcessed++
		result.Clicks += clicks
		result.Captures += captures

		if hour.Add(time.Hour).Before(to) {
			timer := time.NewTimer(backfillHourDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return result, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return result, nil
}

func (s *LinkAnalyticsIngester) processHour(ctx context.Context, from, to time.Time) (int64, int64, error) {
	counts, err := s.processor.Process(ctx, from, to)
	if err != nil {
		return 0, 0, err
	}
	if err := s.writer.UpsertHourly(ctx, counts); err != nil {
		return 0, 0, err
	}

	var clicks int64
	for _, count := range counts {
		clicks += count.Count
	}

	var captures int64
	if s.captures != nil {
		captureCounts, err := s.captures.Processor.Process(ctx, from, to)
		if err != nil {
			return 0, 0, err
		}
		if err := s.captures.Writer.UpsertHourly(ctx, captureCounts); err != nil {
			return 0, 0, err
		}
		for _, count := range captureCounts {
			captures += count.Count
		}
	}
	return clicks, captures, nil
}
