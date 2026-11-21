package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// LinkFilter contains the short-link list filters.
type LinkFilter struct {
	OrgID  string
	Query  string
	Tags   []string
	Limit  int
	Offset int
}

// LinkUpdate contains the optional fields for a link update. UpdatedAt is
// always set.
type LinkUpdate struct {
	URL       *string
	Title     *string
	Tags      *[]string
	Disabled  *bool
	UpdatedAt time.Time
}

// LinkDAO persists short links. Deleted links are tombstones: they keep their
// slug reserved and are excluded from reads.
type LinkDAO interface {
	Insert(ctx context.Context, link entities.Link) (entities.Link, error)
	FindByID(ctx context.Context, orgID, id string) (entities.Link, error)
	FindBySlug(ctx context.Context, slug string) (entities.Link, error)
	List(ctx context.Context, filter LinkFilter) ([]entities.Link, error)
	Update(ctx context.Context, orgID, id string, update LinkUpdate) (entities.Link, error)
	Delete(ctx context.Context, orgID, id string, deletedAt time.Time) (entities.Link, error)
}

// LinkLookupDAO resolves link slugs without excluding disabled or deleted
// records. It supports historical analytics attribution.
type LinkLookupDAO interface {
	FindBySlugs(ctx context.Context, slugs []string) ([]entities.Link, error)
}
