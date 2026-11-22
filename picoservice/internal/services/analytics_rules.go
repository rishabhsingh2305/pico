package services

import (
	"crypto/rand"
	"net/url"
	"regexp"
	"strings"

	"picoservice/internal/validation"
)

const (
	analyticsKeyLength         = 16
	analyticsKeyAlphabet       = "abcdefghijkmnpqrstuvwxyz23456789"
	analyticsDescriptionMax    = 280
	analyticsEventMaxLength    = 64
	analyticsEventMaxCount     = 20
	analyticsQueryMaxLength    = 64
	analyticsReferrerMaxLength = 255
)

var analyticsEventPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

func normalizeAnalyticsEvents(events []string) ([]string, error) {
	seen := make(map[string]struct{}, len(events))
	items := make([]string, 0, len(events))
	for _, raw := range events {
		event := strings.ToLower(strings.TrimSpace(raw))
		if event == "" {
			continue
		}
		if !analyticsEventPattern.MatchString(event) {
			return nil, &validation.Error{Field: "events", Message: "events must use lowercase letters, digits, dots, underscores, or hyphens"}
		}
		if _, ok := seen[event]; ok {
			continue
		}
		seen[event] = struct{}{}
		items = append(items, event)
	}
	if len(items) == 0 {
		return nil, &validation.Error{Field: "events", Message: "at least one event is required"}
	}
	if len(items) > analyticsEventMaxCount {
		return nil, &validation.Error{Field: "events", Message: "events must contain at most 20 items"}
	}
	return items, nil
}

func normalizeAnalyticsEvent(raw string) (string, bool) {
	event := strings.ToLower(strings.TrimSpace(raw))
	if event == "" || !analyticsEventPattern.MatchString(event) {
		return "", false
	}
	return event, true
}

func validateAnalyticsDescription(raw string) (string, error) {
	description := strings.TrimSpace(raw)
	if len(description) > analyticsDescriptionMax {
		return "", &validation.Error{Field: "description", Message: "description must be at most 280 characters"}
	}
	return description, nil
}

func randomAnalyticsKey() (string, error) {
	bytes := make([]byte, analyticsKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	key := make([]byte, analyticsKeyLength)
	for i, value := range bytes {
		key[i] = analyticsKeyAlphabet[int(value)%len(analyticsKeyAlphabet)]
	}
	return string(key), nil
}

func analyticsDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if ua == "" {
		return "unknown"
	}
	if analyticsIsBot(ua) {
		return "bot"
	}
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") || strings.Contains(ua, "kindle") || strings.Contains(ua, "silk") {
		return "tablet"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android") {
		return "mobile"
	}
	return "desktop"
}

func analyticsIsBot(ua string) bool {
	ua = strings.ToLower(ua)
	for _, marker := range []string{
		"bot", "crawler", "spider", "preview", "headless", "curl", "wget",
		"python-requests", "facebookexternalhit", "slack", "discord",
		"telegram", "whatsapp", "linkedin", "twitterbot", "google-inspectiontool",
	} {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

func analyticsCountryCode(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 2 {
		return "unknown"
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return "unknown"
		}
	}
	return code
}

func analyticsReferrerHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "direct"
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "direct"
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if len(host) > analyticsReferrerMaxLength {
		host = host[:analyticsReferrerMaxLength]
	}
	return host
}

func truncateAnalyticsQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > analyticsQueryMaxLength {
		return raw[:analyticsQueryMaxLength]
	}
	return raw
}
