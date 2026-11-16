package entities

import (
	"errors"
	"time"
)

const (
	// FeatureShare covers share-link management.
	FeatureShare = "share"

	// CredentialKindShare is a public or resource-scoped share link.
	CredentialKindShare = "share"
	// CredentialKindAPIKey is a machine credential.
	CredentialKindAPIKey = "api_key"

	// CredentialResourceOrg scopes a credential to an organization.
	CredentialResourceOrg = "org"
	// CredentialResourceProject scopes a credential to one project.
	CredentialResourceProject = "project"
	// CredentialResourceExpense scopes a credential to one expense.
	CredentialResourceExpense = "expense"

	// CredentialStatusActive means the credential is neither revoked nor
	// expired.
	CredentialStatusActive = "active"
	// CredentialStatusExpired means the credential passed its expiry.
	CredentialStatusExpired = "expired"
	// CredentialStatusRevoked means the credential was revoked.
	CredentialStatusRevoked = "revoked"
)

// ErrCredentialNotFound is returned when a credential does not exist.
var ErrCredentialNotFound = errors.New("credential not found")

// ShareScope is the resource a share credential may read.
type ShareScope struct {
	ResourceType string
	ResourceID   string
	ProjectID    string
}

// AllowsProject reports whether scope may read projectID.
func (s ShareScope) AllowsProject(projectID string) bool {
	if s.ResourceType == CredentialResourceOrg {
		return true
	}
	return s.ProjectID != "" && s.ProjectID == projectID
}

// AllowsExpense reports whether scope may read expense.
func (s ShareScope) AllowsExpense(expense Expense) bool {
	switch s.ResourceType {
	case CredentialResourceOrg:
		return true
	case CredentialResourceProject:
		return s.ProjectID != "" && expense.ProjectID == s.ProjectID
	case CredentialResourceExpense:
		return s.ResourceID == expense.ID
	default:
		return false
	}
}

// Credential is a share link or API key. The raw token is never stored.
type Credential struct {
	ID           string
	Kind         string
	OrgID        string
	ResourceType string
	ResourceID   string
	ProjectID    string
	Scopes       []string
	Name         string
	TokenHash    string
	CreatedBy    string
	CreatedAt    time.Time
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	LastUsedAt   *time.Time
}

// Active reports whether the credential is usable at now.
func (c Credential) Active(now time.Time) bool {
	if c.RevokedAt != nil {
		return false
	}
	return c.ExpiresAt == nil || c.ExpiresAt.After(now)
}

// Status returns active, expired, or revoked.
func (c Credential) Status(now time.Time) string {
	switch {
	case c.RevokedAt != nil:
		return CredentialStatusRevoked
	case c.ExpiresAt != nil && !c.ExpiresAt.After(now):
		return CredentialStatusExpired
	default:
		return CredentialStatusActive
	}
}

// ShareScope returns the resource scope for a share credential.
func (c Credential) ShareScope() ShareScope {
	return ShareScope{
		ResourceType: c.ResourceType,
		ResourceID:   c.ResourceID,
		ProjectID:    c.ProjectID,
	}
}

// HasScope reports whether the credential grants permission.
func (c Credential) HasScope(permission string) bool {
	for _, scope := range c.Scopes {
		if scope == permission {
			return true
		}
	}
	return false
}
