package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// OrgUpdate contains optional organization changes.
type OrgUpdate struct {
	Name      *string
	Tags      *[]string
	UpdatedAt time.Time
}

// OrgDAO persists organizations.
type OrgDAO interface {
	Insert(ctx context.Context, org entities.Org) (entities.Org, error)
	FindByID(ctx context.Context, id string) (entities.Org, error)
	FindByIDs(ctx context.Context, ids []string, tags []string) ([]entities.Org, error)
	Update(ctx context.Context, id string, update OrgUpdate) (entities.Org, error)
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
	OwnerID(ctx context.Context, orgID string) (ownerID string, found bool, err error)
}
