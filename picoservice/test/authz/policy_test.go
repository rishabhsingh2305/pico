package authztest

import (
	"strings"
	"testing"

	"picoservice/internal/authz"
	"picoservice/internal/entities"
)

func TestLoad(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	wantRoles := []string{
		"expense.admin", "expense.editor", "expense.viewer",
		"member.admin", "member.viewer",
		"org.admin", "org.viewer",
		"project.admin", "project.editor", "project.viewer",
		"share.admin", "share.viewer",
	}
	assertStrings(t, policy.Roles(), wantRoles)
	assertStrings(t, policy.Features(), []string{"expense", "member", "org", "project", "share"})

	tests := []struct {
		method string
		path   string
		want   string
		scope  authz.Scope
	}{
		{method: "POST", path: "/auth/login", want: "auth.login", scope: authz.ScopePublic},
		{method: "GET", path: "/users/me", want: "users.me", scope: authz.ScopeUser},
		{method: "GET", path: "/expenses", want: "expenses.list", scope: authz.ScopeOrg},
		{method: "POST", path: "/expenses", want: "expenses.create", scope: authz.ScopeOrg},
		{method: "GET", path: "/expenses/summary", want: "expenses.summary", scope: authz.ScopeOrg},
		{method: "PATCH", path: "/orgs/{orgID}", want: "orgs.update", scope: authz.ScopeOrg},
		{method: "DELETE", path: "/orgs/{orgID}/members/{memberUserID}", want: "members.remove", scope: authz.ScopeOrg},
	}

	for _, tt := range tests {
		endpoint, ok := policy.Endpoint(tt.method, tt.path)
		if !ok {
			t.Fatalf("expected %s %s to be defined", tt.method, tt.path)
		}
		if endpoint.Permission != tt.want {
			t.Fatalf("expected permission %q, got %q", tt.want, endpoint.Permission)
		}
		if endpoint.Scope != tt.scope {
			t.Fatalf("expected scope %q, got %q", tt.scope, endpoint.Scope)
		}
	}
}

func TestValidRole(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	tests := []struct {
		feature string
		role    string
		want    bool
	}{
		{feature: entities.FeatureOrg, role: entities.FeatureRoleViewer, want: true},
		{feature: entities.FeatureOrg, role: entities.FeatureRoleAdmin, want: true},
		{feature: entities.FeatureOrg, role: entities.FeatureRoleEditor, want: false},
		{feature: entities.FeatureMember, role: entities.FeatureRoleEditor, want: false},
		{feature: entities.FeatureExpense, role: entities.FeatureRoleEditor, want: true},
		{feature: "unknown", role: entities.FeatureRoleViewer, want: false},
	}
	for _, tt := range tests {
		if got := policy.ValidRole(tt.feature, tt.role); got != tt.want {
			t.Fatalf("%s.%s: expected %v, got %v", tt.feature, tt.role, tt.want, got)
		}
	}
}

func TestEndpointFollowsGetForHead(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	endpoint, ok := policy.Endpoint("HEAD", "/expenses")
	if !ok {
		t.Fatal("expected HEAD to follow the GET policy")
	}
	if endpoint.Permission != "expenses.list" {
		t.Fatalf("expected permission %q, got %q", "expenses.list", endpoint.Permission)
	}
}

func TestEndpointNormalizesTrailingSlash(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	endpoint, ok := policy.Endpoint("GET", "/projects/")
	if !ok {
		t.Fatal("expected the trailing slash to be normalized")
	}
	if endpoint.Permission != "projects.list" {
		t.Fatalf("expected permission %q, got %q", "projects.list", endpoint.Permission)
	}
}

func TestAllow(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	tests := []struct {
		name       string
		role       string
		permission string
		want       bool
	}{
		{name: "owner wildcard", role: entities.RoleWildcard, permission: "expenses.delete", want: true},
		{name: "owner unknown permission", role: entities.RoleWildcard, permission: "expenses.unknown", want: false},
		{name: "expense editor writes", role: "expense.editor", permission: "expenses.create", want: true},
		{name: "expense editor cannot delete", role: "expense.editor", permission: "expenses.delete", want: false},
		{name: "expense admin deletes", role: "expense.admin", permission: "expenses.delete", want: true},
		{name: "expense viewer reads", role: "expense.viewer", permission: "expenses.list", want: true},
		{name: "expense viewer cannot write", role: "expense.viewer", permission: "expenses.create", want: false},
		{name: "member admin manages", role: "member.admin", permission: "members.add", want: true},
		{name: "member viewer reads", role: "member.viewer", permission: "members.list", want: true},
		{name: "member viewer cannot manage", role: "member.viewer", permission: "members.add", want: false},
		{name: "project editor writes", role: "project.editor", permission: "projects.create", want: true},
		{name: "project viewer cannot write", role: "project.viewer", permission: "projects.create", want: false},
		{name: "org admin updates", role: "org.admin", permission: "orgs.update", want: true},
		{name: "org viewer cannot update", role: "org.viewer", permission: "orgs.update", want: false},
		{name: "cross feature denied", role: "project.editor", permission: "expenses.create", want: false},
		{name: "unknown role", role: "content.admin", permission: "expenses.list", want: false},
		{name: "empty role", role: "", permission: "expenses.list", want: false},
		{name: "public permission is not granted", role: "expense.editor", permission: "auth.login", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.Allow(tt.role, tt.permission); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRolePermissions(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	permissions, ok := policy.RolePermissions("expense.viewer")
	if !ok {
		t.Fatal("expected expense.viewer")
	}
	for _, permission := range permissions {
		if strings.HasPrefix(permission, "expenses.create") || strings.HasPrefix(permission, "expenses.delete") {
			t.Fatalf("viewer must not write, got %q", permission)
		}
	}
}

func TestParseRejectsUnknownPermission(t *testing.T) {
	parseError(t, routesWithExpenses(), `
features:
  expense:
    viewer:
      - expenses.missing
`)
}

func TestParseRejectsWildcard(t *testing.T) {
	parseError(t, routesWithExpenses(), `
features:
  expense:
    viewer:
      - "*"
`)
}

func TestParseRejectsNonOrgPermission(t *testing.T) {
	parseError(t, `
endpoints:
  users:
    me:
      method: GET
      path: /users/me
      scope: user
`, `
features:
  expense:
    viewer:
      - users.me
`)
}

func TestParseRejectsWrongFeaturePrefix(t *testing.T) {
	parseError(t, `
endpoints:
  expenses:
    list:
      method: GET
      path: /expenses
      scope: org
  projects:
    list:
      method: GET
      path: /projects
      scope: org
`, `
features:
  expense:
    viewer:
      - projects.list
`)
}

func TestParseRejectsDuplicatePermission(t *testing.T) {
	parseError(t, routesWithExpenses(), `
features:
  expense:
    viewer:
      - expenses.list
      - expenses.list
`)
}

func TestParseRejectsCoverageGap(t *testing.T) {
	parseError(t, `
endpoints:
  expenses:
    list:
      method: GET
      path: /expenses
      scope: org
    create:
      method: POST
      path: /expenses
      scope: org
`, `
features:
  expense:
    viewer:
      - expenses.list
`)
}

func TestParseRejectsHierarchyViolation(t *testing.T) {
	parseError(t, `
endpoints:
  expenses:
    list:
      method: GET
      path: /expenses
      scope: org
    create:
      method: POST
      path: /expenses
      scope: org
`, `
features:
  expense:
    viewer:
      - expenses.list
    editor:
      - expenses.create
`)
}

func TestParseRejectsUnknownFeature(t *testing.T) {
	parseError(t, routesWithExpenses(), `
features:
  content:
    viewer:
      - expenses.list
`)
}

func TestParseRejectsMissingViewer(t *testing.T) {
	parseError(t, routesWithExpenses(), `
features:
  expense:
    admin:
      - expenses.list
`)
}

func TestParseRejectsUnknownRole(t *testing.T) {
	parseError(t, routesWithExpenses(), `
features:
  expense:
    viewer:
      - expenses.list
    owner:
      - expenses.list
`)
}

func TestParseRejectsDuplicateRoute(t *testing.T) {
	parseError(t, `
endpoints:
  projects:
    list:
      method: GET
      path: /projects
      scope: org
    read:
      method: GET
      path: /projects/
      scope: org
`, minimalFeatures("projects.list"))
}

func TestParseRejectsInvalidScope(t *testing.T) {
	parseError(t, `
endpoints:
  projects:
    list:
      method: GET
      path: /projects
      scope: member
`, minimalFeatures("projects.list"))
}

func TestParseRejectsHead(t *testing.T) {
	parseError(t, `
endpoints:
  projects:
    list:
      method: HEAD
      path: /projects
      scope: org
`, minimalFeatures("projects.list"))
}

func TestParseRejectsUnknownField(t *testing.T) {
	parseError(t, `
endpoints:
  projects:
    list:
      method: GET
      path: /projects
      scope: org
      permission: projects.read
`, minimalFeatures("projects.list"))
}

func TestParseRejectsEmptyPolicy(t *testing.T) {
	if _, err := authz.Parse([]byte("endpoints: {}"), []byte("features: {}")); err == nil {
		t.Fatal("expected an empty policy error")
	}
}

func routesWithExpenses() string {
	return `
endpoints:
  expenses:
    list:
      method: GET
      path: /expenses
      scope: org
`
}

func minimalFeatures(permission string) string {
	return `
features:
  project:
    viewer:
      - ` + permission + `
`
}

func parseError(t *testing.T, routes, features string) {
	t.Helper()
	if _, err := authz.Parse([]byte(routes), []byte(features)); err == nil {
		t.Fatal("expected a policy error")
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
