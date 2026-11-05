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

// OrgOwnerLookup reports an organization's owner. Found is false when the org
// does not exist.
type OrgOwnerLookup interface {
	OwnerID(ctx context.Context, orgID string) (ownerID string, found bool, err error)
}

// UserLookup reports whether a user exists.
type UserLookup interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// CreateMembershipInput contains the fields needed to add a member.
type CreateMembershipInput struct {
	ActorID string
	OrgID   string
	UserID  string
}

// ListMembershipsInput contains pagination fields for listing an
// organization's members.
type ListMembershipsInput struct {
	ActorID string
	OrgID   string
	Limit   int
	Offset  int
}

// MembershipService applies membership validation and business rules.
type MembershipService struct {
	store dao.MembershipDAO
	orgs  OrgOwnerLookup
	users UserLookup
}

// NewMembershipService returns a membership service backed by store, orgs, and
// users.
func NewMembershipService(store dao.MembershipDAO, orgs OrgOwnerLookup, users UserLookup) *MembershipService {
	return &MembershipService{store: store, orgs: orgs, users: users}
}

// Add adds a member to an organization. Only the owner may add members.
func (s *MembershipService) Add(ctx context.Context, in CreateMembershipInput) (entities.Membership, error) {
	orgID, userID, err := validateIDs(in.OrgID, in.UserID)
	if err != nil {
		return entities.Membership{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	ownerID, found, err := s.orgs.OwnerID(ctx, orgID)
	if err != nil {
		return entities.Membership{}, fmt.Errorf("get org owner: %w", err)
	}
	if !found {
		return entities.Membership{}, entities.ErrOrgNotFound
	}
	if ownerID != in.ActorID {
		return entities.Membership{}, entities.ErrForbidden
	}
	if userID == ownerID {
		return entities.Membership{}, entities.ErrAlreadyMember
	}

	exists, err := s.users.Exists(ctx, userID)
	if err != nil {
		return entities.Membership{}, fmt.Errorf("check user: %w", err)
	}
	if !exists {
		return entities.Membership{}, entities.ErrUserNotFound
	}

	alreadyMember, err := s.store.Exists(ctx, orgID, userID)
	if err != nil {
		return entities.Membership{}, fmt.Errorf("check membership: %w", err)
	}
	if alreadyMember {
		return entities.Membership{}, entities.ErrAlreadyMember
	}

	membership := entities.Membership{
		OrgID:     orgID,
		UserID:    userID,
		Role:      entities.RoleMember,
		CreatedAt: time.Now().UTC(),
	}
	created, err := s.store.Insert(ctx, membership)
	if err != nil {
		return entities.Membership{}, fmt.Errorf("add member: %w", err)
	}
	created.Role = entities.RoleMember
	return created, nil
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
// actor is a member before this is called.
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
	for i := range items {
		if items[i].UserID == ownerID {
			items[i].Role = entities.RoleOwner
		} else {
			items[i].Role = entities.RoleMember
		}
	}
	return items, nil
}

// Remove removes a member from an organization. Only the owner may remove
// members, and the owner cannot remove themselves.
func (s *MembershipService) Remove(ctx context.Context, actorID, orgID, userID string) error {
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
	if ownerID != actorID {
		return entities.ErrForbidden
	}
	if userID == ownerID {
		return entities.ErrCannotRemoveOwner
	}

	if err := s.store.Delete(ctx, orgID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// Exists reports whether a user belongs to an organization. It is used by
// identity middleware.
func (s *MembershipService) Exists(ctx context.Context, orgID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	exists, err := s.store.Exists(ctx, orgID, userID)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return exists, nil
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
