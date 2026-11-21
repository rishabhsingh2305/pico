package apptest

import (
	"strings"
	"testing"
	"time"

	"picoservice/internal/app"
)

func TestAnalyticsConfigFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		pullZone string
		apiKey   string
		shortURL string
		wantHost string
		wantErr  string
	}{
		{
			name:     "valid",
			pullZone: "1234",
			apiKey:   "bunny-key",
			shortURL: "https://link.uppixl.com/",
			wantHost: "link.uppixl.com",
		},
		{
			name:     "missing pull zone",
			apiKey:   "bunny-key",
			shortURL: "https://link.uppixl.com",
			wantErr:  "BUNNY_PULL_ZONE_ID is not set",
		},
		{
			name:     "invalid pull zone",
			pullZone: "abc",
			apiKey:   "bunny-key",
			shortURL: "https://link.uppixl.com",
			wantErr:  "BUNNY_PULL_ZONE_ID must be a positive integer",
		},
		{
			name:     "missing api key",
			pullZone: "1234",
			shortURL: "https://link.uppixl.com",
			wantErr:  "BUNNY_API_KEY is not set",
		},
		{
			name:     "missing short url",
			pullZone: "1234",
			apiKey:   "bunny-key",
			wantErr:  "SHORT_BASE_URL must be an absolute http or https URL",
		},
		{
			name:     "short url with path",
			pullZone: "1234",
			apiKey:   "bunny-key",
			shortURL: "https://link.uppixl.com/s",
			wantErr:  "SHORT_BASE_URL must not contain a path, query, or fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BUNNY_PULL_ZONE_ID", tt.pullZone)
			t.Setenv("BUNNY_API_KEY", tt.apiKey)
			t.Setenv("SHORT_BASE_URL", tt.shortURL)

			cfg, err := app.AnalyticsConfigFromEnv()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := cfg.BunnyPullZoneID, int64(1234); got != want {
				t.Fatalf("expected pull zone %d, got %d", want, got)
			}
			if got, want := cfg.ShortLinkHost, tt.wantHost; got != want {
				t.Fatalf("expected host %q, got %q", want, got)
			}
			if cfg.Job.MaxLookback > 72*time.Hour {
				t.Fatalf("unexpected lookback: %v", cfg.Job.MaxLookback)
			}
		})
	}
}
