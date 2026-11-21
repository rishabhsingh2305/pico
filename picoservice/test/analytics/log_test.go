package analyticstest

import (
	"testing"
	"time"

	"picoservice/internal/analytics"
)

func TestNormalizeHost(t *testing.T) {
	if got, want := analytics.NormalizeHost(" Link.Uppixl.com. "), "link.uppixl.com"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRequestKeyPrefersRequestID(t *testing.T) {
	entry := analytics.LogEntry{
		Timestamp: time.Now().UTC(),
		RequestID: "request-1",
		Host:      "link.uppixl.com",
		Path:      "/s/launch",
	}
	if got, want := analytics.RequestKey(entry), "request-1"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRequestKeyFallbackIsStable(t *testing.T) {
	entry := analytics.LogEntry{
		Timestamp:  time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
		Host:       "link.uppixl.com",
		Path:       "/s/launch",
		StatusCode: 302,
		RemoteIP:   "192.0.2.1",
		UserAgent:  "Mozilla/5.0",
	}
	first := analytics.RequestKey(entry)
	second := analytics.RequestKey(entry)
	if first == "" || first != second {
		t.Fatalf("expected a stable fallback key, got %q and %q", first, second)
	}
}
