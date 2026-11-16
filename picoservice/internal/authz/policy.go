// Package authz loads the declarative route and feature-role policy and answers
// two questions: which unit permission a route requires, and whether a
// feature-scoped role grants a unit permission. It has no HTTP or database
// dependencies.
package authz

import (
	"fmt"
	"sort"
	"strings"

	"picoservice/internal/entities"
)

// Scope describes how a route is authorized.
type Scope string

const (
	// ScopePublic routes need no identity.
	ScopePublic Scope = "public"
	// ScopeUser routes need a valid identity but no organization role.
	ScopeUser Scope = "user"
	// ScopeOrg routes need a valid identity, an organization role, and a unit
	// permission.
	ScopeOrg Scope = "org"
)

// Endpoint is one route policy entry.
type Endpoint struct {
	// Permission is the unit permission derived from the endpoint key path,
	// for example "expenses.list".
	Permission string
	Method     string
	Path       string
	Scope      Scope
}

// Policy is an immutable route and feature-role policy loaded at startup.
type Policy struct {
	endpoints     map[string]Endpoint
	roles         map[string]map[string]struct{} // "expense.editor" -> permissions
	features      map[string][]string            // "expense" -> declared roles
	permissionSet map[string]struct{}
}

// Endpoint returns the policy entry for a method and route pattern. HEAD
// requests follow the GET policy.
func (p *Policy) Endpoint(method, path string) (Endpoint, bool) {
	method = normalizeMethod(method)
	path = normalizePath(path)
	if method == "HEAD" {
		if endpoint, ok := p.endpoints[routeKey("GET", path)]; ok {
			return endpoint, true
		}
	}
	endpoint, ok := p.endpoints[routeKey(method, path)]
	return endpoint, ok
}

// Allow reports whether role grants permission. The owner wildcard grants every
// defined permission.
func (p *Policy) Allow(role, permission string) bool {
	if role == entities.RoleWildcard {
		_, ok := p.permissionSet[permission]
		return ok
	}
	permissions, ok := p.roles[role]
	if !ok {
		return false
	}
	_, ok = permissions[permission]
	return ok
}

// RolePermissions returns the sorted unit permissions granted to a
// feature-scoped role such as "expense.editor".
func (p *Policy) RolePermissions(role string) ([]string, bool) {
	permissions, ok := p.roles[role]
	if !ok {
		return nil, false
	}
	items := make([]string, 0, len(permissions))
	for permission := range permissions {
		items = append(items, permission)
	}
	sort.Strings(items)
	return items, true
}

// Endpoints returns every route policy entry sorted by method and path.
func (p *Policy) Endpoints() []Endpoint {
	items := make([]Endpoint, 0, len(p.endpoints))
	for _, endpoint := range p.endpoints {
		items = append(items, endpoint)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Method != items[j].Method {
			return items[i].Method < items[j].Method
		}
		return items[i].Path < items[j].Path
	})
	return items
}

// Roles returns every declared feature-scoped role, sorted. Role names have the
// form "<feature>.<role>", for example "expense.editor".
func (p *Policy) Roles() []string {
	items := make([]string, 0, len(p.roles))
	for role := range p.roles {
		items = append(items, role)
	}
	sort.Strings(items)
	return items
}

// Features returns every declared feature, sorted.
func (p *Policy) Features() []string {
	items := make([]string, 0, len(p.features))
	for feature := range p.features {
		items = append(items, feature)
	}
	sort.Strings(items)
	return items
}

// ValidRole reports whether features declares role for feature.
func (p *Policy) ValidRole(feature, role string) bool {
	roles, ok := p.features[feature]
	if !ok {
		return false
	}
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func endpointPermission(group, action string) string {
	return group + "." + action
}

func routeKey(method, path string) string {
	return normalizeMethod(method) + " " + normalizePath(path)
}

func normalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func (s Scope) valid() bool {
	switch s {
	case ScopePublic, ScopeUser, ScopeOrg:
		return true
	default:
		return false
	}
}

func (e Endpoint) validate() error {
	if e.Permission == "" {
		return fmt.Errorf("permission is required")
	}
	if !validMethod(e.Method) {
		return fmt.Errorf("%s: unsupported method %q", e.Permission, e.Method)
	}
	if e.Method == "HEAD" {
		return fmt.Errorf("%s: do not declare HEAD; it follows GET", e.Permission)
	}
	if e.Path == "" || !strings.HasPrefix(e.Path, "/") {
		return fmt.Errorf("%s: path must start with /", e.Permission)
	}
	if strings.Contains(e.Path, "?") {
		return fmt.Errorf("%s: path must not contain a query string", e.Permission)
	}
	if !e.Scope.valid() {
		return fmt.Errorf("%s: unsupported scope %q", e.Permission, e.Scope)
	}
	return nil
}

func validMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
