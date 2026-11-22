package analyticsservicetest

import (
	"context"
	"testing"
	"time"

	"picoservice/internal/analytics"
	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

type fakeSourceStore struct {
	inserted entities.AnalyticsSource
}

func (s *fakeSourceStore) Insert(_ context.Context, source entities.AnalyticsSource) (entities.AnalyticsSource, error) {
	s.inserted = source
	source.ID = "source-1"
	return source, nil
}

func (s *fakeSourceStore) FindByID(context.Context, string, string) (entities.AnalyticsSource, error) {
	return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
}

func (s *fakeSourceStore) FindByKey(context.Context, string) (entities.AnalyticsSource, error) {
	return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
}

func (s *fakeSourceStore) FindByKeys(context.Context, []string) ([]entities.AnalyticsSource, error) {
	return []entities.AnalyticsSource{}, nil
}

func (s *fakeSourceStore) List(context.Context, dao.AnalyticsSourceFilter) ([]entities.AnalyticsSource, error) {
	return []entities.AnalyticsSource{}, nil
}

func (s *fakeSourceStore) Update(context.Context, string, string, dao.AnalyticsSourceUpdate) (entities.AnalyticsSource, error) {
	return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
}

func (s *fakeSourceStore) Delete(context.Context, string, string) error {
	return nil
}

func TestCreateAnalyticsSource(t *testing.T) {
	store := &fakeSourceStore{}
	service := services.NewAnalyticsSourceService(store)

	source, err := service.Create(context.Background(), services.CreateAnalyticsSourceInput{
		OrgID:       "org-1",
		Name:        " Homepage CTA ",
		Description: " top button ",
		Events:      []string{"View", "click", "view"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if source.ID != "source-1" || len(source.Key) != 16 {
		t.Fatalf("unexpected source: %+v", source)
	}
	if source.Name != "Homepage CTA" || source.Description != "top button" {
		t.Fatalf("unexpected normalized fields: %+v", source)
	}
	if len(source.Events) != 2 || source.Events[0] != "view" || source.Events[1] != "click" {
		t.Fatalf("unexpected events: %v", source.Events)
	}
}

func TestCreateAnalyticsSourceDefaultsView(t *testing.T) {
	store := &fakeSourceStore{}
	service := services.NewAnalyticsSourceService(store)

	source, err := service.Create(context.Background(), services.CreateAnalyticsSourceInput{
		OrgID: "org-1",
		Name:  "Email",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(source.Events) != 1 || source.Events[0] != "view" {
		t.Fatalf("expected default view event, got %v", source.Events)
	}
}

func TestCreateAnalyticsSourceRejectsInvalidEvent(t *testing.T) {
	service := services.NewAnalyticsSourceService(&fakeSourceStore{})
	_, err := service.Create(context.Background(), services.CreateAnalyticsSourceInput{
		OrgID:  "org-1",
		Name:   "Email",
		Events: []string{"bad event"},
	})
	var validationErr *validation.Error
	if err == nil || !asValidation(err, &validationErr) || validationErr.Field != "events" {
		t.Fatalf("expected events validation error, got %v", err)
	}
}

type fakeCaptureReader struct {
	stats  dao.AnalyticsStats
	totals []dao.AnalyticsSourceTotal
	filter dao.AnalyticsCaptureFilter
	bulk   dao.AnalyticsBulkFilter
}

func (r *fakeCaptureReader) Stats(_ context.Context, filter dao.AnalyticsCaptureFilter) (dao.AnalyticsStats, error) {
	r.filter = filter
	return r.stats, nil
}

func (r *fakeCaptureReader) SumBySources(_ context.Context, filter dao.AnalyticsBulkFilter) ([]dao.AnalyticsSourceTotal, error) {
	r.bulk = filter
	return r.totals, nil
}

func TestAnalyticsStatsReader(t *testing.T) {
	reader := &fakeCaptureReader{
		stats: dao.AnalyticsStats{
			Total:  12,
			Series: []dao.AnalyticsSeriesPoint{{Bucket: time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC), Count: 12}},
			Events: []dao.AnalyticsEventTotal{{Event: "click", Count: 12}},
		},
	}
	service := services.NewAnalyticsStatsReader(reader)

	result, err := service.Stats(context.Background(), services.AnalyticsStatsInput{
		OrgID:     "org-1",
		SourceID:  "source-1",
		From:      time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		To:        time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		Interval:  "day",
		Event:     "CLICK",
		Countries: []string{"in", "US"},
		Devices:   []string{"mobile"},
	})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if result.Total != 12 || len(result.Series) != 1 || len(result.Events) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if reader.filter.Event != "click" {
		t.Fatalf("expected normalized event, got %q", reader.filter.Event)
	}
	if len(reader.filter.Countries) != 2 || reader.filter.Countries[0] != "IN" {
		t.Fatalf("unexpected countries: %v", reader.filter.Countries)
	}
}

type fakeLogSource struct {
	pages []analytics.LogPage
}

func (s *fakeLogSource) List(_ context.Context, _ analytics.LogQuery) (analytics.LogPage, error) {
	if len(s.pages) == 0 {
		return analytics.LogPage{}, nil
	}
	page := s.pages[0]
	s.pages = s.pages[1:]
	return page, nil
}

type fakeSourceResolver struct {
	sources []entities.AnalyticsSource
}

func (r *fakeSourceResolver) FindByKeys(_ context.Context, keys []string) ([]entities.AnalyticsSource, error) {
	out := make([]entities.AnalyticsSource, 0)
	for _, source := range r.sources {
		for _, key := range keys {
			if source.Key == key {
				out = append(out, source)
			}
		}
	}
	return out, nil
}

func TestAnalyticsCaptureProcessor(t *testing.T) {
	from := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	source := &fakeLogSource{pages: []analytics.LogPage{
		{
			Entries: []analytics.LogEntry{
				{RequestID: "1", Host: "link.uppixl.com", Path: "/p/abc123?event=click", StatusCode: 200, CountryCode: "IN", UserAgent: "iPhone Mobile", Referer: "https://twitter.com/x"},
				{RequestID: "1", Host: "link.uppixl.com", Path: "/p/abc123?event=click", StatusCode: 200, CountryCode: "IN", UserAgent: "iPhone Mobile", Referer: "https://twitter.com/x"},
				{RequestID: "2", Host: "link.uppixl.com", Path: "/p/abc123", StatusCode: 200, CountryCode: "US", UserAgent: "Mozilla/5.0 Chrome", Referer: ""},
				{RequestID: "3", Host: "link.uppixl.com", Path: "/p/unknown?event=click", StatusCode: 200, CountryCode: "IN", UserAgent: "Mozilla/5.0"},
				{RequestID: "4", Host: "link.uppixl.com", Path: "/p/abc123?event=other", StatusCode: 200, CountryCode: "IN", UserAgent: "Mozilla/5.0"},
				{RequestID: "5", Host: "other.example.com", Path: "/p/abc123?event=click", StatusCode: 200},
				{RequestID: "6", Host: "link.uppixl.com", Path: "/p/abc123?event=click", StatusCode: 404},
			},
		},
	}}
	resolver := &fakeSourceResolver{sources: []entities.AnalyticsSource{{
		ID:     "source-1",
		OrgID:  "org-1",
		Key:    "abc123",
		Events: []string{"view", "click"},
	}}}
	processor := services.NewAnalyticsCaptureProcessor(source, resolver, "link.uppixl.com", 100)

	counts, err := processor.Process(context.Background(), from, from.Add(time.Hour))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("expected two dimension groups, got %+v", counts)
	}
	var click, view bool
	for _, count := range counts {
		if count.Event == "click" {
			click = true
			if count.CountryCode != "IN" || count.Device != "mobile" || count.ReferrerHost != "twitter.com" || count.Count != 1 {
				t.Fatalf("unexpected click count: %+v", count)
			}
		}
		if count.Event == "view" {
			view = true
			if count.CountryCode != "US" || count.Device != "desktop" || count.ReferrerHost != "direct" || count.Count != 1 {
				t.Fatalf("unexpected view count: %+v", count)
			}
		}
	}
	if !click || !view {
		t.Fatalf("expected click and view, got %+v", counts)
	}
}

func asValidation(err error, target **validation.Error) bool {
	validationErr, ok := err.(*validation.Error)
	if ok {
		*target = validationErr
	}
	return ok
}
