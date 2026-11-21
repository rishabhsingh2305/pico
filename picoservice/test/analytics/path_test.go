package analyticstest

import (
	"testing"

	"picoservice/internal/analytics"
)

func TestParseShortLinkPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		slug   string
		wantOK bool
	}{
		{name: "simple", path: "/s/launch", slug: "launch", wantOK: true},
		{name: "query", path: "/s/launch?utm=email", slug: "launch", wantOK: true},
		{name: "uppercase", path: "/s/Launch", slug: "launch", wantOK: true},
		{name: "nested", path: "/s/launch/extra", wantOK: false},
		{name: "empty", path: "/s/", wantOK: false},
		{name: "other path", path: "/pixel/launch", wantOK: false},
		{name: "invalid escape", path: "/s/%zz", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, ok := analytics.ParseShortLinkPath(tt.path)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if slug != tt.slug {
				t.Fatalf("expected slug %q, got %q", tt.slug, slug)
			}
		})
	}
}
