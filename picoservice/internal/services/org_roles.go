package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

// maxTokenOrganizations caps the organization role map carried by an access
// token. Each organization carries one role string per feature.
const maxTokenOrganizations = 20

// MembershipRoleStore reads a user's memberships for token claims.
type MembershipRoleStore interface {
	ListByUser(ctx context.Context, userID string, limit int) ([]entities.Membership, error)
}

// OrgOwnersLookup returns organization owners keyed by organization ID.
type OrgOwnersLookup interface {
	OwnerIDs(ctx context.Context, orgIDs []string) (map[string]string, error)
}

// OrgRoleResolver projects memberships and ownership into the feature role map
// written into an access token.
type OrgRoleResolver struct {
	memberships MembershipRoleStore
	orgs        OrgOwnersLookup
}

// NewOrgRoleResolver returns a resolver backed by memberships and org owners.
func NewOrgRoleResolver(memberships MembershipRoleStore, orgs OrgOwnersLookup) *OrgRoleResolver {
	return &OrgRoleResolver{memberships: memberships, orgs: orgs}
}

// RolesForUser returns the feature role list for each organization. Owners get
// ["*"]; members get strings such as "expense.editor".
func (r *OrgRoleResolver) RolesForUser(ctx context.Context, userID string) (map[string][]string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, &validation.Error{Field: "userId", Message: "userId is required"}
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	items, err := r.memberships.ListByUser(ctx, userID, maxTokenOrganizations)
	if err != nil {
		return nil, fmt.Errorf("list user memberships: %w", err)
	}
	if len(items) == 0 {
		return map[string][]string{}, nil
	}

	orgIDs := make([]string, 0, len(items))
	for _, item := range items {
		orgIDs = append(orgIDs, item.OrgID)
	}
	owners, err := r.orgs.OwnerIDs(ctx, orgIDs)
	if err != nil {
		return nil, fmt.Errorf("get org owners: %w", err)
	}

	roles := make(map[string][]string, len(items))
	for _, item := range items {
		if owners[item.OrgID] == userID {
			roles[item.OrgID] = []string{entities.RoleWildcard}
			continue
		}
		featureRoles := featureRoleNames(item.Roles)
		if len(featureRoles) == 0 {
			continue
		}
		roles[item.OrgID] = featureRoles
	}
	return roles, nil
}

func featureRoleNames(roles map[string]string) []string {
	if len(roles) == 0 {
		return nil
	}
	names := make([]string, 0, len(roles))
	for feature, role := range roles {
		feature = strings.TrimSpace(feature)
		role = strings.TrimSpace(role)
		if feature == "" || role == "" {
			continue
		}
		names = append(names, feature+"."+role)
	}
	sort.Strings(names)
	return names
}
