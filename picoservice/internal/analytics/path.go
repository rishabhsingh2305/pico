package analytics

import (
	"net/url"
	"strings"
)

const shortLinkPathPrefix = "/s/"

// ParseShortLinkPath extracts one slug from a public short-link request path.
// Query strings are ignored and nested paths are rejected.
func ParseShortLinkPath(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	path := parsed.Path
	if !strings.HasPrefix(path, shortLinkPathPrefix) {
		return "", false
	}

	slug := strings.TrimPrefix(path, shortLinkPathPrefix)
	if slug == "" || strings.Contains(slug, "/") {
		return "", false
	}
	return strings.ToLower(slug), true
}
