package dao

import (
	"context"

	"picoservice/internal/entities"
)

// MembershipDAO persists memberships.
type MembershipDAO interface {
	Insert(ctx context.Context, membership entities.Membership) (entities.Membership, error)
	AddOwner(ctx context.Context, orgID, userID string) error
	ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]entities.Membership, error)
	ListOrgIDsByUser(ctx context.Context, userID string, limit, offset int) ([]string, error)
	Exists(ctx context.Context, orgID, userID string) (bool, error)
	Delete(ctx context.Context, orgID, userID string) error
}
