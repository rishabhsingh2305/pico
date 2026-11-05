package entities

import "time"

// Org is an organization record. UserID is the owner.
type Org struct {
	ID        string
	Name      string
	UserID    string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}
