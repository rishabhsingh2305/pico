package linkanalyticstest

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"picoservice/internal/analytics"
	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

type fakeLogSource struct {
	listFunc func(context.Context, analytics.LogQuery) (analytics.LogPage, error)
}

func (s fakeLogSource) List(ctx context.Context, query analytics.LogQuery) (analytics.LogPage, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, query)
	}
	return analytics.LogPage{}, nil
}

type fakeLinkResolver struct {
	findFunc func(context.Context, []string) ([]entities.Link, error)
}

func (r fakeLinkResolver) FindBySlugs(ctx context.Context, slugs []string) ([]entities.Link, error) {
	if r.findFunc != nil {
		return r.findFunc(ctx, slugs)
	}
	return []entities.Link{}, nil
}

type fakeClickWriter struct {
	calls [][]dao.LinkClickCount
	err   error
}

func (w *fakeClickWriter) UpsertHourly(_ context.Context, counts []dao.LinkClickCount) error {
	w.calls = append(w.calls, counts)
	return w.err
}

type fakeStateStore struct {
	state      dao.LinkAnalyticsState
	acquired   bool
	acquireErr error
	advances   []time.Time
	released   bool
	initial    time.Time
}

func (s *fakeStateStore) Acquire(
	_ context.Context,
	_ int64,
	_, _, initialHour time.Time,
) (dao.LinkAnalyticsState, bool, error) {
	s.initial = initialHour
	if s.acquireErr != nil {
		return dao.LinkAnalyticsState{}, false, s.acquireErr
	}
	return s.state, s.acquired, nil
}

func (s *fakeStateStore) Advance(_ context.Context, _ int64, hour time.Time) error {
	s.advances = append(s.advances, hour)
	if hour.After(s.state.LastCompletedHour) {
		s.state.LastCompletedHour = hour
	}
	return nil
}

func (s *fakeStateStore) Release(_ context.Context, _ int64, _ time.Time) error {
	s.released = true
	return nil
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 22, 10, 20, 0, 0, time.UTC)
}

func TestLinkHourProcessorCountsAndDeduplicates(t *testing.T) {
	from := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	source := fakeLogSource{
		listFunc: func(_ context.Context, query analytics.LogQuery) (analytics.LogPage, error) {
			if !query.From.Equal(from) || !query.To.Equal(from.Add(time.Hour)) {
				t.Fatalf("unexpected range: %v %v", query.From, query.To)
			}
			if query.Host != "link.uppixl.com" || query.Status != "2xx,3xx" {
				t.Fatalf("unexpected query: %+v", query)
			}
			return analytics.LogPage{
				Entries: []analytics.LogEntry{
					{RequestID: "request-1", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302, CacheStatus: "MISS"},
					{RequestID: "request-1", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302, CacheStatus: "HIT"},
					{RequestID: "request-2", Host: "link.uppixl.com", Path: "/s/launch?utm=email", StatusCode: 302, CacheStatus: "HIT"},
					{RequestID: "request-3", Host: "other.uppixl.com", Path: "/s/launch", StatusCode: 302},
					{RequestID: "request-4", Host: "link.uppixl.com", Path: "/s/missing", StatusCode: 302},
					{RequestID: "request-5", Host: "link.uppixl.com", Path: "/p/launch", StatusCode: 200},
					{RequestID: "request-6", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 404},
				},
			}, nil
		},
	}
	resolver := fakeLinkResolver{
		findFunc: func(_ context.Context, slugs []string) ([]entities.Link, error) {
			if len(slugs) != 2 || slugs[0] != "launch" || slugs[1] != "missing" {
				t.Fatalf("unexpected slugs: %v", slugs)
			}
			return []entities.Link{{ID: "link-1", OrgID: "org-1", Slug: "launch"}}, nil
		},
	}

	processor := services.NewLinkHourProcessor(source, resolver, "Link.Uppixl.com", 100)
	counts, err := processor.Process(context.Background(), from, from.Add(time.Hour))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(counts) != 1 {
		t.Fatalf("expected one count, got %+v", counts)
	}
	if counts[0].LinkID != "link-1" || counts[0].OrgID != "org-1" || counts[0].Count != 2 {
		t.Fatalf("unexpected count: %+v", counts[0])
	}
	if !counts[0].HourStart.Equal(from) {
		t.Fatalf("expected hour %v, got %v", from, counts[0].HourStart)
	}
}

func TestLinkHourProcessorPaginates(t *testing.T) {
	from := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	source := fakeLogSource{
		listFunc: func(_ context.Context, query analytics.LogQuery) (analytics.LogPage, error) {
			switch query.Offset {
			case 0:
				return analytics.LogPage{
					Entries: []analytics.LogEntry{{
						RequestID: "request-1", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302,
					}},
					NextOffset: 1,
					HasMore:    true,
				}, nil
			case 1:
				return analytics.LogPage{
					Entries: []analytics.LogEntry{{
						RequestID: "request-2", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302,
					}},
					NextOffset: 2,
					HasMore:    false,
				}, nil
			default:
				t.Fatalf("unexpected offset: %d", query.Offset)
				return analytics.LogPage{}, nil
			}
		},
	}
	resolver := fakeLinkResolver{
		findFunc: func(context.Context, []string) ([]entities.Link, error) {
			return []entities.Link{{ID: "link-1", OrgID: "org-1", Slug: "launch"}}, nil
		},
	}

	processor := services.NewLinkHourProcessor(source, resolver, "link.uppixl.com", 1)
	counts, err := processor.Process(context.Background(), from, from.Add(time.Hour))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(counts) != 1 || counts[0].Count != 2 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
}

func TestLinkHourProcessorRejectsStalledPagination(t *testing.T) {
	source := fakeLogSource{
		listFunc: func(context.Context, analytics.LogQuery) (analytics.LogPage, error) {
			return analytics.LogPage{
				Entries: []analytics.LogEntry{{
					RequestID: "request-1", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302,
				}},
				NextOffset: 0,
				HasMore:    true,
			}, nil
		},
	}
	processor := services.NewLinkHourProcessor(source, fakeLinkResolver{}, "link.uppixl.com", 1)
	_, err := processor.Process(context.Background(),
		time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 22, 11, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("expected a pagination error")
	}
}

func TestLinkHourProcessorEmpty(t *testing.T) {
	processor := services.NewLinkHourProcessor(fakeLogSource{}, fakeLinkResolver{}, "link.uppixl.com", 100)
	counts, err := processor.Process(context.Background(),
		time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 22, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(counts) != 0 {
		t.Fatalf("expected no counts, got %+v", counts)
	}
}

func TestLinkAnalyticsIngesterProcessesCurrentAndPreviousHours(t *testing.T) {
	writer := &fakeClickWriter{}
	state := &fakeStateStore{
		state:    dao.LinkAnalyticsState{LastCompletedHour: time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)},
		acquired: true,
	}
	source := fakeLogSource{
		listFunc: func(context.Context, analytics.LogQuery) (analytics.LogPage, error) {
			return analytics.LogPage{Entries: []analytics.LogEntry{{
				RequestID: "request-1", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302,
			}}}, nil
		},
	}
	processor := services.NewLinkHourProcessor(source, fakeLinkResolver{
		findFunc: func(context.Context, []string) ([]entities.Link, error) {
			return []entities.Link{{ID: "link-1", OrgID: "org-1", Slug: "launch"}}, nil
		},
	}, "link.uppixl.com", 100)
	ingester := services.NewLinkAnalyticsIngester(processor, writer, state, services.LinkAnalyticsJobConfig{
		PullZoneID:      1,
		LeaseDuration:   time.Minute,
		SafetyLag:       5 * time.Minute,
		MaxHoursPerRun:  4,
		InitialBackfill: 48 * time.Hour,
		MaxLookback:     71 * time.Hour,
	}, fixedNow)

	result, err := ingester.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !result.CurrentHourProcessed || !result.PreviousHourProcessed {
		t.Fatalf("expected current and previous hours, got %+v", result)
	}
	if result.HoursProcessed != 0 {
		t.Fatalf("expected no catch-up hours, got %+v", result)
	}
	if len(writer.calls) != 2 {
		t.Fatalf("expected two writes, got %d", len(writer.calls))
	}
	if !state.released {
		t.Fatal("expected lease release")
	}
	if result.Clicks != 2 {
		t.Fatalf("expected two clicks, got %d", result.Clicks)
	}
}

func TestLinkAnalyticsIngesterAdvancesEmptyHour(t *testing.T) {
	writer := &fakeClickWriter{}
	state := &fakeStateStore{
		state:    dao.LinkAnalyticsState{LastCompletedHour: time.Date(2026, 11, 22, 9, 0, 0, 0, time.UTC)},
		acquired: true,
	}
	processor := services.NewLinkHourProcessor(fakeLogSource{}, fakeLinkResolver{}, "link.uppixl.com", 100)
	ingester := services.NewLinkAnalyticsIngester(processor, writer, state, services.LinkAnalyticsJobConfig{
		PullZoneID:      1,
		LeaseDuration:   time.Minute,
		SafetyLag:       5 * time.Minute,
		MaxHoursPerRun:  1,
		InitialBackfill: 48 * time.Hour,
		MaxLookback:     71 * time.Hour,
	}, fixedNow)

	result, err := ingester.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.HoursProcessed != 1 {
		t.Fatalf("expected one catch-up hour, got %+v", result)
	}
	if len(state.advances) != 1 || !state.advances[0].Equal(time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected watermark at 10:00, got %v", state.advances)
	}
	if !state.released {
		t.Fatal("expected lease release")
	}
}

func TestLinkAnalyticsIngesterBackfillsRange(t *testing.T) {
	writer := &fakeClickWriter{}
	state := &fakeStateStore{
		state:    dao.LinkAnalyticsState{LastCompletedHour: time.Date(2026, 11, 22, 5, 0, 0, 0, time.UTC)},
		acquired: true,
	}
	source := fakeLogSource{
		listFunc: func(context.Context, analytics.LogQuery) (analytics.LogPage, error) {
			return analytics.LogPage{Entries: []analytics.LogEntry{{
				RequestID: "request-1", Host: "link.uppixl.com", Path: "/s/launch", StatusCode: 302,
			}}}, nil
		},
	}
	processor := services.NewLinkHourProcessor(source, fakeLinkResolver{
		findFunc: func(context.Context, []string) ([]entities.Link, error) {
			return []entities.Link{{ID: "link-1", OrgID: "org-1", Slug: "launch"}}, nil
		},
	}, "link.uppixl.com", 100)
	ingester := services.NewLinkAnalyticsIngester(processor, writer, state, services.LinkAnalyticsJobConfig{
		PullZoneID:      1,
		LeaseDuration:   time.Minute,
		SafetyLag:       5 * time.Minute,
		MaxHoursPerRun:  12,
		InitialBackfill: 48 * time.Hour,
		MaxLookback:     71 * time.Hour,
	}, fixedNow)

	result, err := ingester.Backfill(
		context.Background(),
		time.Date(2026, 11, 21, 21, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 21, 22, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.HoursProcessed != 1 || result.Clicks != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("expected one write, got %d", len(writer.calls))
	}
	if len(state.advances) != 0 {
		t.Fatalf("backfill must not advance the watermark, got %v", state.advances)
	}
	if !state.released {
		t.Fatal("expected lease release")
	}
}

func TestLinkAnalyticsIngesterSkipsLockedRun(t *testing.T) {
	writer := &fakeClickWriter{}
	state := &fakeStateStore{acquired: false}
	ingester := services.NewLinkAnalyticsIngester(
		services.NewLinkHourProcessor(fakeLogSource{}, fakeLinkResolver{}, "link.uppixl.com", 100),
		writer,
		state,
		services.LinkAnalyticsJobConfig{
			PullZoneID:      1,
			LeaseDuration:   time.Minute,
			SafetyLag:       5 * time.Minute,
			MaxHoursPerRun:  1,
			InitialBackfill: 48 * time.Hour,
			MaxLookback:     71 * time.Hour,
		},
		fixedNow,
	)

	result, err := ingester.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !result.Skipped || len(writer.calls) != 0 || state.released {
		t.Fatalf("unexpected locked result: %+v, writes=%d, released=%v", result, len(writer.calls), state.released)
	}
}

type fakeStatsReader struct {
	totals []dao.LinkClickTotal
	err    error
	filter dao.LinkStatsFilter
}

func (r *fakeStatsReader) SumByLinks(_ context.Context, filter dao.LinkStatsFilter) ([]dao.LinkClickTotal, error) {
	r.filter = filter
	return r.totals, r.err
}

func TestLinkStatsReader(t *testing.T) {
	reader := &fakeStatsReader{totals: []dao.LinkClickTotal{{LinkID: "link-1", Clicks: 12}}}
	service := services.NewLinkStatsReader(reader)

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	stats, err := service.Stats(context.Background(), services.LinkStatsInput{
		OrgID:   "org-1",
		LinkIDs: []string{"link-2", "link-1", "link-1"},
		From:    from,
		To:      to,
	})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.Items) != 1 || stats.Items[0].LinkID != "link-1" || stats.Items[0].Clicks != 12 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(reader.filter.LinkIDs) != 2 {
		t.Fatalf("expected deduplicated ids, got %v", reader.filter.LinkIDs)
	}
}

func TestLinkStatsReaderValidatesInput(t *testing.T) {
	service := services.NewLinkStatsReader(&fakeStatsReader{})
	now := time.Now().UTC()
	tooManyIDs := make([]string, services.LinkStatsMaxIDs+1)
	for i := range tooManyIDs {
		tooManyIDs[i] = "link-" + strconv.Itoa(i)
	}

	tests := []struct {
		name  string
		input services.LinkStatsInput
		field string
	}{
		{
			name:  "missing ids",
			input: services.LinkStatsInput{OrgID: "org-1", From: now.Add(-time.Hour), To: now},
			field: "ids",
		},
		{
			name:  "too many ids",
			input: services.LinkStatsInput{OrgID: "org-1", LinkIDs: tooManyIDs, From: now.Add(-time.Hour), To: now},
			field: "ids",
		},
		{
			name:  "invalid range",
			input: services.LinkStatsInput{OrgID: "org-1", LinkIDs: []string{"link-1"}, From: now, To: now.Add(-time.Hour)},
			field: "from",
		},
		{
			name:  "range too large",
			input: services.LinkStatsInput{OrgID: "org-1", LinkIDs: []string{"link-1"}, From: now.Add(-services.LinkStatsMaxRange - time.Hour), To: now},
			field: "to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Stats(context.Background(), tt.input)
			var validationErr *validation.Error
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("expected field %q, got %q", tt.field, validationErr.Field)
			}
		})
	}
}
