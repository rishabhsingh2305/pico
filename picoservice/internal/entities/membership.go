package entities

import "time"

const (
	// RoleOwner identifies the organization owner.
	RoleOwner = "owner"
	// RoleMember identifies a regular organization member.
	RoleMember = "member"
)

// Membership is a user's relation to an organization. Role is derived from the
// organization owner and is not persisted.
type Membership struct {
	ID        string
	OrgID     string
	UserID    string
	Role      string
	CreatedAt time.Time
}
