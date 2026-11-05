package entities

import "time"

// Project is a project record.
type Project struct {
	ID        string
	OrgID     string
	UserID    string
	Name      string
	Currency  string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}
