package services

import (
	"crypto/rand"
	"net/url"
	"regexp"
	"strings"

	"picoservice/internal/validation"
)

const (
	linkSlugLength       = 7
	maxSlugAttempts      = 5
	linkSlugMinLength    = 3
	linkSlugMaxLength    = 32
	linkURLMaxLength     = 2048
	linkTitleMaxLength   = 120
	linkQueryMaxLength   = 64
	linkSlugAlphabet     = "abcdefghijkmnpqrstuvwxyz23456789"
	linkSlugAlphabetSize = len(linkSlugAlphabet)
)

var linkSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,30}[a-z0-9])$`)

func validateLinkURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", &validation.Error{Field: "url", Message: "url is required"}
	}
	if len(raw) > linkURLMaxLength {
		return "", &validation.Error{Field: "url", Message: "url must be at most 2048 characters"}
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", &validation.Error{Field: "url", Message: "url must be a valid http or https URL"}
	}

	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Host == "" || parsed.Hostname() == "" {
		return "", &validation.Error{Field: "url", Message: "url must be a valid http or https URL"}
	}
	if parsed.User != nil {
		return "", &validation.Error{Field: "url", Message: "url must not include credentials"}
	}

	parsed.Scheme = scheme
	return parsed.String(), nil
}

func validateLinkTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if len(title) > linkTitleMaxLength {
		return "", &validation.Error{Field: "title", Message: "title must be at most 120 characters"}
	}
	return title, nil
}

func validateLinkSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if len(slug) < linkSlugMinLength || len(slug) > linkSlugMaxLength {
		return "", &validation.Error{Field: "slug", Message: "slug must be 3 to 32 characters"}
	}
	if !linkSlugPattern.MatchString(slug) {
		return "", &validation.Error{Field: "slug", Message: "slug must use lowercase letters, digits, and hyphens"}
	}
	return slug, nil
}

func randomLinkSlug() (string, error) {
	bytes := make([]byte, linkSlugLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	slug := make([]byte, linkSlugLength)
	for i, value := range bytes {
		slug[i] = linkSlugAlphabet[int(value)%linkSlugAlphabetSize]
	}
	return string(slug), nil
}
