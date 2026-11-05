package services

import (
	"slices"
	"strings"

	"picoservice/internal/validation"
)

const (
	tagMaxLength = 32
	maxTags      = 20
)

// normalizeTagFilter trims, deduplicates, and sorts filter tags. Empty values
// are ignored because filters are optional.
func normalizeTagFilter(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	filter := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		filter = append(filter, tag)
	}
	slices.Sort(filter)
	return filter
}

// normalizeTags trims, validates, deduplicates, and sorts stored tags.
func normalizeTags(tags []string) ([]string, error) {
	if len(tags) > maxTags {
		return nil, &validation.Error{Field: "tags", Message: "tags must contain at most 20 items"}
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			return nil, &validation.Error{Field: "tags", Message: "tags must not contain empty values"}
		}
		if len(tag) > tagMaxLength {
			return nil, &validation.Error{Field: "tags", Message: "each tag must be at most 32 characters"}
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	slices.Sort(normalized)
	return normalized, nil
}
