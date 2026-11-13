package services

import (
	"slices"
	"strings"

	"picoservice/internal/validation"
)

const (
	tagMaxLength  = 32
	maxTags       = 20
	maxProjectIDs = 100
)

// normalizeProjectIDs trims, deduplicates, and sorts filter project IDs. Empty
// values are ignored because the filter is optional.
func normalizeProjectIDs(projectIDs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(projectIDs))
	normalized := make([]string, 0, len(projectIDs))
	for _, raw := range projectIDs {
		projectID := strings.TrimSpace(raw)
		if projectID == "" {
			continue
		}
		if _, ok := seen[projectID]; ok {
			continue
		}
		seen[projectID] = struct{}{}
		normalized = append(normalized, projectID)
	}
	if len(normalized) > maxProjectIDs {
		return nil, &validation.Error{Field: "projectIds", Message: "projectIds must contain at most 100 items"}
	}
	slices.Sort(normalized)
	return normalized, nil
}

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
