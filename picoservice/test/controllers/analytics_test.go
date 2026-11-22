package controllerstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"picoservice/internal/controllers"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestAnalyticsPixel(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p/abc123", nil)

	controllers.NewAnalyticsPixel().Pixel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "image/gif"; got != want {
		t.Fatalf("expected content type %q, got %q", want, got)
	}
	if rec.Header().Get("Cache-Control") != "no-store, max-age=0" {
		t.Fatalf("expected uncached pixel response, got %q", rec.Header().Get("Cache-Control"))
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected GIF body")
	}
}

func TestAnalyticsCreate(t *testing.T) {
	sources := &fakes.AnalyticsSourceService{
		CreateFunc: func(_ context.Context, in services.CreateAnalyticsSourceInput) (entities.AnalyticsSource, error) {
			if in.OrgID != "org-1" || in.Name != "Homepage" {
				t.Fatalf("unexpected input: %+v", in)
			}
			return entities.AnalyticsSource{
				ID:     "source-1",
				OrgID:  "org-1",
				Name:   in.Name,
				Key:    "abc123",
				Events: []string{"view"},
			}, nil
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/analytics/sources?orgId=org-1", strings.NewReader(`{"name":"Homepage","events":["view"]}`))
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewAnalyticsController(testLogger(), sources, &fakes.AnalyticsStatsService{}, "https://link.uppixl.com").Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["pixelUrl"], "https://link.uppixl.com/p/abc123"; got != want {
		t.Fatalf("expected pixelUrl %q, got %v", want, got)
	}
}

func TestAnalyticsStats(t *testing.T) {
	sources := &fakes.AnalyticsSourceService{}
	stats := &fakes.AnalyticsStatsService{
		StatsFunc: func(_ context.Context, in services.AnalyticsStatsInput) (services.AnalyticsStatsResult, error) {
			if in.SourceID != "source-1" || in.Event != "click" {
				t.Fatalf("unexpected input: %+v", in)
			}
			return services.AnalyticsStatsResult{
				Total: 12,
				Series: []services.AnalyticsSeriesItem{{
					Bucket: time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC),
					Count:  12,
				}},
			}, nil
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/analytics/sources/source-1/stats?orgId=org-1&event=click&bot=false", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"sourceID": "source-1"})

	controllers.NewAnalyticsController(testLogger(), sources, stats, "https://link.uppixl.com").Stats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total != 12 {
		t.Fatalf("expected total 12, got %d", body.Total)
	}
}

func TestAnalyticsStatsRejectsInvalidBot(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/analytics/sources/source-1/stats?orgId=org-1&bot=maybe", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"sourceID": "source-1"})

	controllers.NewAnalyticsController(
		testLogger(),
		&fakes.AnalyticsSourceService{},
		&fakes.AnalyticsStatsService{},
		"https://link.uppixl.com",
	).Stats(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "bot must be true or false")
}
