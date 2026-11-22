package entities

import "time"

const (
	// FeatureOrg covers organization settings.
	FeatureOrg = "org"
	// FeatureMember covers organization membership management.
	FeatureMember = "member"
	// FeatureProject covers projects.
	FeatureProject = "project"
	// FeatureExpense covers expenses.
	FeatureExpense = "expense"
	// FeatureLink covers short links.
	FeatureLink = "link"
	// FeatureAnalytics covers pixel analytics sources and reads.
	FeatureAnalytics = "analytics"

	// FeatureRoleViewer can read a feature.
	FeatureRoleViewer = "viewer"
	// FeatureRoleEditor can read and write a feature.
	FeatureRoleEditor = "editor"
	// FeatureRoleAdmin can read, write, and manage a feature.
	FeatureRoleAdmin = "admin"

	// RoleWildcard is written into an access token for an organization the
	// user owns. It is never stored on a membership.
	RoleWildcard = "*"
)

// ValidFeature reports whether feature is a known feature name.
func ValidFeature(feature string) bool {
	switch feature {
	case FeatureOrg, FeatureMember, FeatureProject, FeatureExpense, FeatureLink, FeatureShare, FeatureAnalytics:
		return true
	default:
		return false
	}
}

// ValidFeatureRole reports whether role is a feature role name.
func ValidFeatureRole(role string) bool {
	switch role {
	case FeatureRoleViewer, FeatureRoleEditor, FeatureRoleAdmin:
		return true
	default:
		return false
	}
}

// DefaultMemberRoles returns the roles used for a member added without explicit
// roles. It matches the previous org.editor behavior.
func DefaultMemberRoles() map[string]string {
	return map[string]string{
		FeatureOrg:       FeatureRoleViewer,
		FeatureMember:    FeatureRoleViewer,
		FeatureProject:   FeatureRoleEditor,
		FeatureExpense:   FeatureRoleEditor,
		FeatureLink:      FeatureRoleViewer,
		FeatureAnalytics: FeatureRoleViewer,
	}
}

// ViewerRoles returns the read-only role map used by the previous org.viewer
// role.
func ViewerRoles() map[string]string {
	return map[string]string{
		FeatureOrg:       FeatureRoleViewer,
		FeatureProject:   FeatureRoleViewer,
		FeatureExpense:   FeatureRoleViewer,
		FeatureLink:      FeatureRoleViewer,
		FeatureAnalytics: FeatureRoleViewer,
	}
}

// LegacyMemberRoles maps a stored pre-feature role to the new role map.
// Unknown values are treated as read-only.
func LegacyMemberRoles(role string) map[string]string {
	switch role {
	case "org.viewer":
		return ViewerRoles()
	case "org.editor", "member", "owner", "":
		return DefaultMemberRoles()
	default:
		return ViewerRoles()
	}
}

// Membership is a user's relation to an organization. Roles maps a feature to
// the user's role for that feature. Owner is derived from the organization and
// is not stored.
type Membership struct {
	ID        string
	OrgID     string
	UserID    string
	Roles     map[string]string
	Owner     bool
	CreatedAt time.Time
}
