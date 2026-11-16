package services

import (
	"fmt"
	"strings"

	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

func (s *MembershipService) normalizeRoles(roles map[string]string) (map[string]string, error) {
	if roles == nil {
		return s.defaultRoles(), nil
	}
	if len(roles) == 0 {
		return nil, &validation.Error{Field: "roles", Message: "roles must include at least one feature"}
	}

	normalized := make(map[string]string, len(roles))
	for feature, role := range roles {
		feature = strings.TrimSpace(feature)
		role = strings.TrimSpace(role)
		if feature == "" {
			return nil, &validation.Error{Field: "roles", Message: "feature name is required"}
		}
		if role == "" {
			return nil, &validation.Error{Field: "roles", Message: "role is required for feature " + feature}
		}
		if !s.validRole(feature, role) {
			return nil, &validation.Error{
				Field:   "roles",
				Message: fmt.Sprintf("feature %q does not allow role %q", feature, role),
			}
		}
		normalized[feature] = role
	}
	return normalized, nil
}

func (s *MembershipService) validRole(feature, role string) bool {
	if s.features != nil {
		return s.features.ValidRole(feature, role)
	}
	return entities.ValidFeature(feature) && entities.ValidFeatureRole(role)
}

func (s *MembershipService) defaultRoles() map[string]string {
	defaults := entities.DefaultMemberRoles()
	filtered := make(map[string]string, len(defaults))
	for feature, role := range defaults {
		if s.validRole(feature, role) {
			filtered[feature] = role
		}
	}
	return filtered
}
