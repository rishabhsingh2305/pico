package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// ProjectFilter contains the project list filters.
type ProjectFilter struct {
	OrgID  string
	Tags   []string
	Limit  int
	Offset int
}

// ProjectUpdate contains the optional project fields to update.
type ProjectUpdate struct {
	Name      *string
	Tags      *[]string
	UpdatedAt time.Time
}

// ProjectDAO persists projects.
type ProjectDAO interface {
	Insert(ctx context.Context, project entities.Project) (entities.Project, error)
	FindByID(ctx context.Context, orgID, id string) (entities.Project, error)
	List(ctx context.Context, filter ProjectFilter) ([]entities.Project, error)
	Update(ctx context.Context, orgID, id string, update ProjectUpdate) (entities.Project, error)
	Exists(ctx context.Context, orgID, projectID string) (bool, error)
}
