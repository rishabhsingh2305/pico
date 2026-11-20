package entities

import "time"

// Link is a public short link managed by an organization. The slug is globally
// unique because redirects are resolved without organization context.
type Link struct {
	ID         string
	OrgID      string
	UserID     string
	Slug       string
	URL        string
	Title      string
	Tags       []string
	DisabledAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Disabled reports whether the link currently redirects.
func (l Link) Disabled() bool {
	return l.DisabledAt != nil
}
