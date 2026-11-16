package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// CredentialFilter selects credentials for a listing.
type CredentialFilter struct {
	OrgID        string
	Kind         string
	ResourceType string
	ResourceIDs  []string
	Active       bool
	Limit        int
	Offset       int
}

// CredentialDAO persists share links and API keys.
type CredentialDAO interface {
	Insert(ctx context.Context, credential entities.Credential) (entities.Credential, error)
	FindByID(ctx context.Context, orgID, id string) (entities.Credential, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (entities.Credential, error)
	List(ctx context.Context, filter CredentialFilter) ([]entities.Credential, error)
	Revoke(ctx context.Context, orgID, id string, revokedAt time.Time) error
}
