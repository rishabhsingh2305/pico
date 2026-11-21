package controllerstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"picoservice/internal/controllers"
	"picoservice/internal/services"
	"picoservice/internal/validation"
	"picoservice/test/fakes"
)

func TestLinkStats(t *testing.T) {
	service := &fakes.LinkAnalyticsService{
		StatsFunc: func(_ context.Context, in services.LinkStatsInput) (services.LinkStats, error) {
			if in.OrgID != "org-1" {
				t.Fatalf("unexpected org: %q", in.OrgID)
			}
			if len(in.LinkIDs) != 2 || in.LinkIDs[0] != "link-1" || in.LinkIDs[1] != "link-2" {
				t.Fatalf("unexpected ids: %v", in.LinkIDs)
			}
			if in.From.IsZero() || in.To.IsZero() || !in.From.Before(in.To) {
				t.Fatalf("unexpected range: %v %v", in.From, in.To)
			}
			return services.LinkStats{Items: []services.LinkStat{
				{LinkID: "link-1", Clicks: 12},
			}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/orgs/org-1/analytics/links?orgId=org-1&ids=link-1,link-2&from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z",
		nil,
	)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewLinkAnalyticsController(testLogger(), service).Stats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []struct {
			LinkID string `json:"linkId"`
			Clicks int64  `json:"clicks"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].LinkID != "link-1" || body.Items[0].Clicks != 12 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestLinkStatsDefaultsRange(t *testing.T) {
	var from, to time.Time
	service := &fakes.LinkAnalyticsService{
		StatsFunc: func(_ context.Context, in services.LinkStatsInput) (services.LinkStats, error) {
			from, to = in.From, in.To
			return services.LinkStats{Items: []services.LinkStat{}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/analytics/links?orgId=org-1&ids=link-1", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewLinkAnalyticsController(testLogger(), service).Stats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if from.IsZero() || to.IsZero() || !to.After(from) {
		t.Fatalf("unexpected default range: %v %v", from, to)
	}
	if got, want := to.Sub(from), 30*24*time.Hour; got != want {
		t.Fatalf("expected %v range, got %v", want, got)
	}
}

func TestLinkStatsRequiresIDs(t *testing.T) {
	service := &fakes.LinkAnalyticsService{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/analytics/links?orgId=org-1", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewLinkAnalyticsController(testLogger(), service).Stats(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "ids is required")
}

func TestLinkStatsValidationError(t *testing.T) {
	service := &fakes.LinkAnalyticsService{
		StatsFunc: func(context.Context, services.LinkStatsInput) (services.LinkStats, error) {
			return services.LinkStats{}, &validation.Error{Field: "from", Message: "from must be before to"}
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/analytics/links?orgId=org-1&ids=link-1", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewLinkAnalyticsController(testLogger(), service).Stats(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "from must be before to")
}
