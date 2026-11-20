package authz

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"picoservice/internal/entities"
)

//go:embed resources/routes.yml resources/features.yml
var policyFiles embed.FS

const (
	routesFile   = "resources/routes.yml"
	featuresFile = "resources/features.yml"
)

// featurePrefixes maps a feature role namespace to the route permission prefix.
// Keeping it explicit catches copy-paste mistakes between features.
var featurePrefixes = map[string]string{
	entities.FeatureOrg:     "orgs",
	entities.FeatureMember:  "members",
	entities.FeatureProject: "projects",
	entities.FeatureExpense: "expenses",
	entities.FeatureLink:    "links",
	entities.FeatureShare:   "shares",
}

type routesDocument struct {
	Endpoints map[string]map[string]endpointDocument `yaml:"endpoints"`
}

type endpointDocument struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
	Scope  string `yaml:"scope"`
}

type featuresDocument struct {
	Features map[string]map[string][]string `yaml:"features"`
}

// Load parses and validates the embedded policy files.
func Load() (*Policy, error) {
	routes, err := policyFiles.ReadFile(routesFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", routesFile, err)
	}
	features, err := policyFiles.ReadFile(featuresFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", featuresFile, err)
	}
	return Parse(routes, features)
}

// Parse validates and compiles the route and feature-role documents.
func Parse(routesYAML, featuresYAML []byte) (*Policy, error) {
	routes, err := decodeRoutes(routesYAML)
	if err != nil {
		return nil, err
	}
	features, err := decodeFeatures(featuresYAML)
	if err != nil {
		return nil, err
	}

	endpoints, err := compileEndpoints(routes)
	if err != nil {
		return nil, err
	}
	roles, declaredFeatures, err := compileFeatures(features, endpoints)
	if err != nil {
		return nil, err
	}

	permissionSet := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		permissionSet[endpoint.Permission] = struct{}{}
	}

	return &Policy{
		endpoints:     endpoints,
		roles:         roles,
		features:      declaredFeatures,
		permissionSet: permissionSet,
	}, nil
}

func decodeRoutes(raw []byte) (routesDocument, error) {
	var document routesDocument
	if err := decodeStrict(routesFile, raw, &document); err != nil {
		return routesDocument{}, err
	}
	if len(document.Endpoints) == 0 {
		return routesDocument{}, fmt.Errorf("%s: endpoints must not be empty", routesFile)
	}
	return document, nil
}

func decodeFeatures(raw []byte) (featuresDocument, error) {
	var document featuresDocument
	if err := decodeStrict(featuresFile, raw, &document); err != nil {
		return featuresDocument{}, err
	}
	if len(document.Features) == 0 {
		return featuresDocument{}, fmt.Errorf("%s: features must not be empty", featuresFile)
	}
	return document, nil
}

func decodeStrict(name string, raw []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("parse %s: multiple YAML documents are not supported", name)
		}
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}

func compileEndpoints(document routesDocument) (map[string]Endpoint, error) {
	endpoints := make(map[string]Endpoint)
	routes := make(map[string]string)
	for group, actions := range document.Endpoints {
		group = strings.TrimSpace(group)
		if group == "" {
			return nil, fmt.Errorf("%s: endpoint group name is required", routesFile)
		}
		if len(actions) == 0 {
			return nil, fmt.Errorf("%s: endpoint group %q is empty", routesFile, group)
		}
		for action, config := range actions {
			action = strings.TrimSpace(action)
			if action == "" {
				return nil, fmt.Errorf("%s: endpoint action in %q is required", routesFile, group)
			}
			endpoint := Endpoint{
				Permission: endpointPermission(group, action),
				Method:     normalizeMethod(config.Method),
				Path:       normalizePath(config.Path),
				Scope:      Scope(strings.ToLower(strings.TrimSpace(config.Scope))),
			}
			if err := endpoint.validate(); err != nil {
				return nil, fmt.Errorf("%s: %w", routesFile, err)
			}

			key := routeKey(endpoint.Method, endpoint.Path)
			if existing, ok := routes[key]; ok {
				return nil, fmt.Errorf(
					"%s: %s %s is declared by both %s and %s",
					routesFile, endpoint.Method, endpoint.Path, existing, endpoint.Permission,
				)
			}
			routes[key] = endpoint.Permission
			endpoints[key] = endpoint
		}
	}
	return endpoints, nil
}

func compileFeatures(document featuresDocument, endpoints map[string]Endpoint) (map[string]map[string]struct{}, map[string][]string, error) {
	scopes := endpointScopes(endpoints)
	compiled := make(map[string]map[string]struct{})
	declared := make(map[string][]string, len(document.Features))

	for feature, roles := range document.Features {
		roleNames, err := compileFeature(feature, roles, scopes, compiled)
		if err != nil {
			return nil, nil, err
		}
		declared[feature] = roleNames
	}

	if err := validateCoverage(compiled, endpoints); err != nil {
		return nil, nil, err
	}
	return compiled, declared, nil
}

func endpointScopes(endpoints map[string]Endpoint) map[string]Scope {
	scopes := make(map[string]Scope, len(endpoints))
	for _, endpoint := range endpoints {
		scopes[endpoint.Permission] = endpoint.Scope
	}
	return scopes
}

func compileFeature(feature string, roles map[string][]string, scopes map[string]Scope, compiled map[string]map[string]struct{}) ([]string, error) {
	feature = strings.TrimSpace(feature)
	prefix, ok := featurePrefixes[feature]
	if !ok {
		return nil, fmt.Errorf("%s: unknown feature %q", featuresFile, feature)
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("%s: feature %q must declare at least one role", featuresFile, feature)
	}
	if _, ok := roles[entities.FeatureRoleViewer]; !ok {
		return nil, fmt.Errorf("%s: feature %q must declare the %q role", featuresFile, feature, entities.FeatureRoleViewer)
	}

	roleNames := make([]string, 0, len(roles))
	for role := range roles {
		roleNames = append(roleNames, role)
	}
	sort.Strings(roleNames)

	for _, role := range roleNames {
		if err := compileFeatureRole(feature, prefix, role, roles[role], scopes, compiled); err != nil {
			return nil, err
		}
	}
	if err := validateRoleHierarchy(feature, compiled); err != nil {
		return nil, err
	}
	return roleNames, nil
}

func compileFeatureRole(feature, prefix, role string, permissions []string, scopes map[string]Scope, compiled map[string]map[string]struct{}) error {
	role = strings.TrimSpace(role)
	if !entities.ValidFeatureRole(role) {
		return fmt.Errorf(
			"%s: feature %q has unsupported role %q; use viewer, editor, or admin",
			featuresFile, feature, role,
		)
	}
	if len(permissions) == 0 {
		return fmt.Errorf("%s: %s.%s must grant at least one permission", featuresFile, feature, role)
	}

	key := featureRole(feature, role)
	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		permission = strings.TrimSpace(permission)
		if permission == "" {
			return fmt.Errorf("%s: %s contains an empty permission", featuresFile, key)
		}
		if permission == entities.RoleWildcard {
			return fmt.Errorf("%s: %q is reserved for the implicit org owner", featuresFile, entities.RoleWildcard)
		}
		if !strings.HasPrefix(permission, prefix+".") {
			return fmt.Errorf(
				"%s: %s grants %q, which does not belong to the %q route group",
				featuresFile, key, permission, prefix,
			)
		}
		scope, ok := scopes[permission]
		if !ok {
			return fmt.Errorf("%s: %s grants unknown permission %q", featuresFile, key, permission)
		}
		if scope != ScopeOrg {
			return fmt.Errorf("%s: %s grants non-org permission %q", featuresFile, key, permission)
		}
		if _, duplicate := granted[permission]; duplicate {
			return fmt.Errorf("%s: %s repeats permission %q", featuresFile, key, permission)
		}
		granted[permission] = struct{}{}
	}
	compiled[key] = granted
	return nil
}

func validateCoverage(compiled map[string]map[string]struct{}, endpoints map[string]Endpoint) error {
	for _, endpoint := range endpoints {
		if endpoint.Scope != ScopeOrg {
			continue
		}
		covered := false
		for _, granted := range compiled {
			if _, ok := granted[endpoint.Permission]; ok {
				covered = true
				break
			}
		}
		if !covered {
			return fmt.Errorf(
				"%s: org permission %q is not granted by any feature role",
				featuresFile, endpoint.Permission,
			)
		}
	}
	return nil
}

func validateRoleHierarchy(feature string, compiled map[string]map[string]struct{}) error {
	viewer, hasViewer := compiled[featureRole(feature, entities.FeatureRoleViewer)]
	editor, hasEditor := compiled[featureRole(feature, entities.FeatureRoleEditor)]
	admin, hasAdmin := compiled[featureRole(feature, entities.FeatureRoleAdmin)]

	if hasEditor {
		if !hasViewer {
			return fmt.Errorf("%s: %s.editor requires a %s.viewer role", featuresFile, feature, feature)
		}
		if err := requireSubset(viewer, editor, feature, entities.FeatureRoleViewer, entities.FeatureRoleEditor); err != nil {
			return err
		}
	}
	if hasAdmin {
		if !hasViewer {
			return fmt.Errorf("%s: %s.admin requires a %s.viewer role", featuresFile, feature, feature)
		}
		if err := requireSubset(viewer, admin, feature, entities.FeatureRoleViewer, entities.FeatureRoleAdmin); err != nil {
			return err
		}
	}
	if hasEditor && hasAdmin {
		if err := requireSubset(editor, admin, feature, entities.FeatureRoleEditor, entities.FeatureRoleAdmin); err != nil {
			return err
		}
	}
	return nil
}

func requireSubset(lower, upper map[string]struct{}, feature, lowerName, upperName string) error {
	for permission := range lower {
		if _, ok := upper[permission]; !ok {
			return fmt.Errorf(
				"%s: %s.%s must include %q, which %s.%s grants",
				featuresFile, feature, upperName, permission, feature, lowerName,
			)
		}
	}
	return nil
}

func featureRole(feature, role string) string {
	return feature + "." + role
}
