package fakes

import (
	"context"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

// MembershipDAO is an in-memory dao.MembershipDAO for tests.
type MembershipDAO struct {
	Memberships map[string]entities.Membership
}

var _ dao.MembershipDAO = (*MembershipDAO)(nil)

// NewMembershipDAO returns an empty in-memory membership store.
func NewMembershipDAO() *MembershipDAO {
	return &MembershipDAO{Memberships: make(map[string]entities.Membership)}
}

// Add inserts a membership with explicit roles. Empty roles use the default
// member map.
func (s *MembershipDAO) Add(orgID, userID string, roles map[string]string) {
	s.AddWithTime(orgID, userID, roles, time.Now().UTC())
}

// AddWithTime inserts a membership with a fixed creation time.
func (s *MembershipDAO) AddWithTime(orgID, userID string, roles map[string]string, createdAt time.Time) {
	if roles == nil {
		roles = entities.DefaultMemberRoles()
	}
	key := membershipKey(orgID, userID)
	s.Memberships[key] = entities.Membership{
		ID:        key,
		OrgID:     orgID,
		UserID:    userID,
		Roles:     cloneRoles(roles),
		CreatedAt: createdAt,
	}
}

// Has reports whether a membership exists.
func (s *MembershipDAO) Has(orgID, userID string) bool {
	_, ok := s.Memberships[membershipKey(orgID, userID)]
	return ok
}

// Get returns a stored membership.
func (s *MembershipDAO) Get(orgID, userID string) (entities.Membership, bool) {
	membership, ok := s.Memberships[membershipKey(orgID, userID)]
	return membership, ok
}

// SetRoles replaces a member's roles.
func (s *MembershipDAO) SetRoles(orgID, userID string, roles map[string]string) bool {
	key := membershipKey(orgID, userID)
	membership, ok := s.Memberships[key]
	if !ok {
		return false
	}
	membership.Roles = cloneRoles(roles)
	s.Memberships[key] = membership
	return true
}

func (s *MembershipDAO) Insert(_ context.Context, membership entities.Membership) (entities.Membership, error) {
	key := membershipKey(membership.OrgID, membership.UserID)
	if _, ok := s.Memberships[key]; ok {
		return entities.Membership{}, entities.ErrAlreadyMember
	}
	membership.ID = key
	if len(membership.Roles) == 0 {
		membership.Roles = entities.DefaultMemberRoles()
	} else {
		membership.Roles = cloneRoles(membership.Roles)
	}
	s.Memberships[key] = membership
	return membership, nil
}

func (s *MembershipDAO) AddOwner(ctx context.Context, orgID, userID string) error {
	_, err := s.Insert(ctx, entities.Membership{
		OrgID:     orgID,
		UserID:    userID,
		Roles:     entities.DefaultMemberRoles(),
		CreatedAt: time.Now().UTC(),
	})
	return err
}

func (s *MembershipDAO) ListByOrg(_ context.Context, orgID string, _, _ int) ([]entities.Membership, error) {
	items := make([]entities.Membership, 0)
	for _, membership := range s.Memberships {
		if membership.OrgID == orgID {
			items = append(items, membership)
		}
	}
	return items, nil
}

func (s *MembershipDAO) ListByUser(_ context.Context, userID string, _ int) ([]entities.Membership, error) {
	items := make([]entities.Membership, 0)
	for _, membership := range s.Memberships {
		if membership.UserID == userID {
			items = append(items, membership)
		}
	}
	return items, nil
}

func (s *MembershipDAO) ListOrgIDsByUser(_ context.Context, userID string, _, _ int) ([]string, error) {
	ids := make([]string, 0)
	for _, membership := range s.Memberships {
		if membership.UserID == userID {
			ids = append(ids, membership.OrgID)
		}
	}
	return ids, nil
}

func (s *MembershipDAO) Exists(_ context.Context, orgID, userID string) (bool, error) {
	return s.Has(orgID, userID), nil
}

func (s *MembershipDAO) UpdateRoles(_ context.Context, orgID, userID string, roles map[string]string) error {
	if !s.SetRoles(orgID, userID, roles) {
		return entities.ErrMembershipNotFound
	}
	return nil
}

func (s *MembershipDAO) Delete(_ context.Context, orgID, userID string) error {
	key := membershipKey(orgID, userID)
	if _, ok := s.Memberships[key]; !ok {
		return entities.ErrMembershipNotFound
	}
	delete(s.Memberships, key)
	return nil
}

func membershipKey(orgID, userID string) string {
	return orgID + "/" + userID
}

func cloneRoles(roles map[string]string) map[string]string {
	cloned := make(map[string]string, len(roles))
	for feature, role := range roles {
		cloned[feature] = role
	}
	return cloned
}
