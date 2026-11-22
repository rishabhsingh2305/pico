package orgrolestest

import (
	"context"
	"errors"
	"testing"

	"picoservice/internal/entities"
	"picoservice/internal/services"
)

func TestRolesForUser(t *testing.T) {
	memberships := fakeMemberships{
		items: map[string][]entities.Membership{
			"user-1": {
				{ID: "m-1", OrgID: "org-owned", UserID: "user-1", Roles: entities.DefaultMemberRoles()},
				{ID: "m-2", OrgID: "org-family", UserID: "user-1", Roles: entities.ViewerRoles()},
				{ID: "m-3", OrgID: "org-work", UserID: "user-1", Roles: entities.DefaultMemberRoles()},
			},
		},
	}
	owners := fakeOwners{owners: map[string]string{"org-owned": "user-1"}}
	resolver := services.NewOrgRoleResolver(memberships, owners)

	roles, err := resolver.RolesForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("roles for user: %v", err)
	}
	assertRoles(t, roles["org-owned"], []string{entities.RoleWildcard})
	assertRoles(t, roles["org-family"], []string{"analytics.viewer", "expense.viewer", "link.viewer", "org.viewer", "project.viewer"})
	assertRoles(t, roles["org-work"], []string{"analytics.viewer", "expense.editor", "link.viewer", "member.viewer", "org.viewer", "project.editor"})
}

func TestRolesForUserSkipsEmptyRoles(t *testing.T) {
	memberships := fakeMemberships{
		items: map[string][]entities.Membership{
			"user-1": {{ID: "m-1", OrgID: "org-1", UserID: "user-1"}},
		},
	}
	resolver := services.NewOrgRoleResolver(memberships, fakeOwners{})

	roles, err := resolver.RolesForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("roles for user: %v", err)
	}
	if _, ok := roles["org-1"]; ok {
		t.Fatal("membership without roles must not grant an organization")
	}
}

func TestRolesForUserRequiresUser(t *testing.T) {
	resolver := services.NewOrgRoleResolver(fakeMemberships{}, fakeOwners{})
	if _, err := resolver.RolesForUser(context.Background(), " "); err == nil {
		t.Fatal("expected a validation error")
	}
}

func TestRolesForUserPropagatesLookupErrors(t *testing.T) {
	resolver := services.NewOrgRoleResolver(fakeMemberships{err: errors.New("db down")}, fakeOwners{})
	if _, err := resolver.RolesForUser(context.Background(), "user-1"); err == nil {
		t.Fatal("expected a membership lookup error")
	}

	memberships := fakeMemberships{items: map[string][]entities.Membership{
		"user-1": {{ID: "m-1", OrgID: "org-1", UserID: "user-1", Roles: entities.ViewerRoles()}},
	}}
	resolver = services.NewOrgRoleResolver(memberships, fakeOwners{err: errors.New("db down")})
	if _, err := resolver.RolesForUser(context.Background(), "user-1"); err == nil {
		t.Fatal("expected an owner lookup error")
	}
}

func assertRoles(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected roles %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected roles %v, got %v", want, got)
		}
	}
}

type fakeMemberships struct {
	items map[string][]entities.Membership
	err   error
}

func (f fakeMemberships) ListByUser(_ context.Context, userID string, _ int) ([]entities.Membership, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items[userID], nil
}

type fakeOwners struct {
	owners map[string]string
	err    error
}

func (f fakeOwners) OwnerIDs(_ context.Context, _ []string) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.owners == nil {
		return map[string]string{}, nil
	}
	return f.owners, nil
}
