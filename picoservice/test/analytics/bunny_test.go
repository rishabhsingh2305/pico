package analyticstest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"picoservice/internal/analytics"
	"picoservice/internal/analytics/bunny"
)

func TestBunnyListBuildsQueryAndParsesEntries(t *testing.T) {
	var gotQuery string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v2/pullzones/123/logs"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("AccessKey")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"data": [{
				"timestamp": "2026-05-08T14:32:11.265Z",
				"requestId": "abc123",
				"host": "link.uppixl.com",
				"path": "/s/launch?utm=email",
				"statusCode": 302,
				"cacheStatus": "HIT",
				"remoteIp": "192.0.2.1",
				"userAgent": "test-agent"
			}],
			"pagination": {"offset": 0, "limit": 100, "returned": 1, "hasMore": true}
		}`)
	}))
	defer server.Close()

	client, err := bunny.New(bunny.Config{
		APIKey:     "secret",
		PullZoneID: 123,
		Endpoint:   server.URL,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	page, err := client.List(context.Background(), analytics.LogQuery{
		From:       time.Date(2026, 5, 8, 14, 0, 0, 0, time.UTC),
		To:         time.Date(2026, 5, 8, 15, 0, 0, 0, time.UTC),
		Host:       "link.uppixl.com",
		PathPrefix: "/s/",
		Status:     "2xx,3xx",
		Limit:      100,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}

	if got, want := gotAuth, "secret"; got != want {
		t.Fatalf("expected api key header %q, got %q", want, got)
	}
	for _, want := range []string{
		"from=2026-05-08T14%3A00%3A00Z",
		"to=2026-05-08T15%3A00%3A00Z",
		"urlContains=%2Fs%2F",
		"status=2xx%2C3xx",
		"order=asc",
		"limit=100",
		"offset=0",
	} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("expected query %q to contain %q", gotQuery, want)
		}
	}

	if len(page.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(page.Entries))
	}
	entry := page.Entries[0]
	if entry.RequestID != "abc123" || entry.Host != "link.uppixl.com" || entry.Path != "/s/launch?utm=email" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.StatusCode != http.StatusFound || entry.CacheStatus != "HIT" {
		t.Fatalf("unexpected status fields: %+v", entry)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("unexpected pagination: %+v", page)
	}
}

func TestBunnyListRetriesRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprint(w, `{"error":{"code":"rate_limited","message":"slow down"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":[],"pagination":{"hasMore":false}}`)
	}))
	defer server.Close()

	client, err := bunny.New(bunny.Config{
		APIKey:      "secret",
		PullZoneID:  123,
		Endpoint:    server.URL,
		RetryDelay:  time.Millisecond,
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.List(context.Background(), analytics.LogQuery{
		From:   time.Date(2026, 5, 8, 14, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 5, 8, 15, 0, 0, 0, time.UTC),
		Limit:  100,
		Status: "3xx",
	})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected two attempts, got %d", attempts)
	}
}

func TestBunnyListReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error":{"code":"logging_disabled","message":"logging is disabled"}}`)
	}))
	defer server.Close()

	client, err := bunny.New(bunny.Config{
		APIKey:     "secret",
		PullZoneID: 123,
		Endpoint:   server.URL,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.List(context.Background(), analytics.LogQuery{
		From:  time.Date(2026, 5, 8, 14, 0, 0, 0, time.UTC),
		To:    time.Date(2026, 5, 8, 15, 0, 0, 0, time.UTC),
		Limit: 100,
	})
	apiErr, ok := err.(*bunny.Error)
	if !ok {
		t.Fatalf("expected *bunny.Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Code != "logging_disabled" {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
}
