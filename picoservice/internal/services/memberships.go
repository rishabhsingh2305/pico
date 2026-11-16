package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

// OrgOwnerLookup reports organization owners.
type OrgOwnerLookup interface {
	OwnerID(ctx context.Context, orgID string) (ownerID string, found bool, err error)
}

// UserLookup reports whether a user exists.
type UserLookup interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// FeatureRoles validates feature-scoped roles against the loaded policy.
type FeatureRoles interface {
	ValidRole(feature, role string) bool
}

// CreateMembershipInput contains the fields needed to add a member. Roles is
// optional and defaults to the legacy editor role map.
type CreateMembershipInput struct {
	OrgID  string
	UserID string
	Roles  map[string]string
}

// ListMembershipsInput contains pagination fields for listing an
// organization's members.
type ListMembershipsInput struct {
	OrgID  string
	Limit  int
	Offset int
}

// MembershipService applies membership validation and business rules.
type MembershipService struct {
	store    dao.MembershipDAO
	orgs     OrgOwnerLookup
	users    UserLookup
	features FeatureRoles
}

// NewMembershipService returns a membership service backed by store, orgs,
// users, and the feature-role policy.
func NewMembershipService(store dao.MembershipDAO, orgs OrgOwnerLookup, users UserLookup, features FeatureRoles) *MembershipService {
	return &MembershipService{store: store, orgs: orgs, users: users, features: features}
}

// Add adds a member to an organization. The router requires members.add.
func (s *MembershipService) Add(ctx context.Context, in CreateMembershipInput) (entities.Membership, error) {
	orgID, userID, err := validateIDs(in.OrgID, in.UserID)
	if err != nil {
		return entities.Membership{}, err
	}
	roles, err := s.normalizeRoles(in.Roles)
	if err != nil {
		return entities.Membership{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := s.checkMembershipTarget(ctx, orgID, userID); err != nil {
		return entities.Membership{}, err
	}

	membership := entities.Membership{
		OrgID:     orgID,
		UserID:    userID,
		Roles:     roles,
		CreatedAt: time.Now().UTC(),
	}
	created, err := s.store.Insert(ctx, membership)
	if err != nil {
		return entities.Membership{}, fmt.Errorf("add member: %w", err)
	}
	created.Roles = roles
	return created, nil
}

func (s *MembershipService) checkMembershipTarget(ctx context.Context, orgID, userID string) error {
	ownerID, found, err := s.orgs.OwnerID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get org owner: %w", err)
	}
	if !found {
		return entities.ErrOrgNotFound
	}
	if userID == ownerID {
		return entities.ErrAlreadyMember
	}

	exists, err := s.users.Exists(ctx, userID)
	if err != nil {
		return fmt.Errorf("check user: %w", err)
	}
	if !exists {
		return entities.ErrUserNotFound
	}

	alreadyMember, err := s.store.Exists(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if alreadyMember {
		return entities.ErrAlreadyMember
	}
	return nil
}

// AddOwner creates the owner's membership during organization creation. It is
// intentionally not exposed through HTTP.
func (s *MembershipService) AddOwner(ctx context.Context, orgID, userID string) error {
	orgID, userID, err := validateIDs(orgID, userID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := s.store.AddOwner(ctx, orgID, userID); err != nil {
		return fmt.Errorf("add owner membership: %w", err)
	}
	return nil
}

// List returns the members of an organization. The router enforces that the
// actor has the members.list permission before this is called.
func (s *MembershipService) List(ctx context.Context, in ListMembershipsInput) ([]entities.Membership, error) {
	orgID := strings.TrimSpace(in.OrgID)
	if orgID == "" {
		return nil, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}

	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	ownerID, found, err := s.orgs.OwnerID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org owner: %w", err)
	}
	if !found {
		return nil, entities.ErrOrgNotFound
	}

	items, err := s.store.ListByOrg(ctx, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	return s.decorateMemberships(items, ownerID), nil
}

func (s *MembershipService) decorateMemberships(items []entities.Membership, ownerID string) []entities.Membership {
	for i := range items {
		items[i].Owner = items[i].UserID == ownerID
		if !items[i].Owner && len(items[i].Roles) == 0 {
			items[i].Roles = s.defaultRoles()
		}
	}
	return items
}

// Remove removes a member from an organization. The router requires
// members.remove, and the owner cannot be removed.
func (s *MembershipService) Remove(ctx context.Context, orgID, userID string) error {
	orgID, userID, err := validateIDs(orgID, userID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	ownerID, found, err := s.orgs.OwnerID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get org owner: %w", err)
	}
	if !found {
		return entities.ErrOrgNotFound
	}
	if userID == ownerID {
		return entities.ErrCannotRemoveOwner
	}

	if err := s.store.Delete(ctx, orgID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// UpdateRoles replaces a member's feature roles. The router requires
// members.update, and the owner's own roles cannot be changed.
func (s *MembershipService) UpdateRoles(ctx context.Context, orgID, userID string, roles map[string]string) error {
	orgID, userID, err := validateIDs(orgID, userID)
	if err != nil {
		return err
	}
	if roles == nil {
		return &validation.Error{Field: "roles", Message: "roles is required"}
	}
	roles, err = s.normalizeRoles(roles)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	ownerID, found, err := s.orgs.OwnerID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get org owner: %w", err)
	}
	if !found {
		return entities.ErrOrgNotFound
	}
	if userID == ownerID {
		return entities.ErrCannotChangeOwner
	}

	if err := s.store.UpdateRoles(ctx, orgID, userID, roles); err != nil {
		return fmt.Errorf("update member roles: %w", err)
	}
	return nil
}

func validateIDs(orgID, userID string) (string, string, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return "", "", &validation.Error{Field: "orgId", Message: "orgId is required"}
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", "", &validation.Error{Field: "userId", Message: "userId is required"}
	}
	return orgID, userID, nil
}
