package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

const (
	shareTokenPrefix     = "pico_share_"
	maxShareLifetime     = 365 * 24 * time.Hour
	maxCredentialNameLen = 120
	maxCredentialIDs     = 100
)

var shareScopes = []string{
	"orgs.get",
	"projects.list",
	"projects.get",
	"expenses.list",
	"expenses.summary",
	"expenses.get",
}

// CredentialOrgLookup returns an organization.
type CredentialOrgLookup interface {
	FindByID(ctx context.Context, id string) (entities.Org, error)
}

// CredentialProjectLookup returns a project scoped to an organization.
type CredentialProjectLookup interface {
	FindByID(ctx context.Context, orgID, id string) (entities.Project, error)
}

// CredentialExpenseLookup returns an expense scoped to an organization.
type CredentialExpenseLookup interface {
	FindByID(ctx context.Context, orgID, id string) (entities.Expense, error)
}

// CreateCredentialInput contains the fields needed to create a share.
type CreateCredentialInput struct {
	ActorID      string
	OrgID        string
	Kind         string
	ResourceType string
	ResourceID   string
	Name         string
	ExpiresAt    *time.Time
}

// CreateCredentialResult contains the stored credential and the one-time raw
// token.
type CreateCredentialResult struct {
	Credential entities.Credential
	Token      string
}

// ListCredentialsInput filters credentials.
type ListCredentialsInput struct {
	OrgID        string
	Kind         string
	ResourceType string
	ResourceIDs  []string
	Active       bool
	Limit        int
	Offset       int
}

// CredentialService creates, resolves, lists, and revokes share links.
type CredentialService struct {
	store    dao.CredentialDAO
	orgs     CredentialOrgLookup
	projects CredentialProjectLookup
	expenses CredentialExpenseLookup
}

// NewCredentialService returns a credential service.
func NewCredentialService(store dao.CredentialDAO, orgs CredentialOrgLookup, projects CredentialProjectLookup, expenses CredentialExpenseLookup) *CredentialService {
	return &CredentialService{store: store, orgs: orgs, projects: projects, expenses: expenses}
}

// Create creates a share credential and returns the raw token once.
func (s *CredentialService) Create(ctx context.Context, in CreateCredentialInput) (CreateCredentialResult, error) {
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = entities.CredentialKindShare
	}
	if kind != entities.CredentialKindShare {
		return CreateCredentialResult{}, &validation.Error{Field: "kind", Message: "kind must be share"}
	}

	orgID := strings.TrimSpace(in.OrgID)
	if orgID == "" {
		return CreateCredentialResult{}, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	resourceType, resourceID, projectID, err := s.resolveResource(ctx, orgID, in.ResourceType, in.ResourceID)
	if err != nil {
		return CreateCredentialResult{}, err
	}

	expiresAt, err := normalizeCredentialExpiry(in.ExpiresAt)
	if err != nil {
		return CreateCredentialResult{}, err
	}

	name := strings.TrimSpace(in.Name)
	if len(name) > maxCredentialNameLen {
		return CreateCredentialResult{}, &validation.Error{Field: "name", Message: "name must be at most 120 characters"}
	}

	rawToken, tokenHash, err := newShareToken()
	if err != nil {
		return CreateCredentialResult{}, err
	}

	credential := entities.Credential{
		Kind:         kind,
		OrgID:        orgID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ProjectID:    projectID,
		Scopes:       scopesForResource(resourceType),
		Name:         name,
		TokenHash:    tokenHash,
		CreatedBy:    in.ActorID,
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    expiresAt,
	}
	created, err := s.store.Insert(ctx, credential)
	if err != nil {
		return CreateCredentialResult{}, fmt.Errorf("create credential: %w", err)
	}
	return CreateCredentialResult{Credential: created, Token: rawToken}, nil
}

func (s *CredentialService) resolveResource(ctx context.Context, orgID, resourceType, resourceID string) (string, string, string, error) {
	resourceType = strings.TrimSpace(resourceType)
	resourceID = strings.TrimSpace(resourceID)

	switch resourceType {
	case entities.CredentialResourceOrg:
		if resourceID != "" && resourceID != orgID {
			return "", "", "", &validation.Error{Field: "resourceId", Message: "resourceId must match the organization"}
		}
		if _, err := s.orgs.FindByID(ctx, orgID); err != nil {
			return "", "", "", err
		}
		return resourceType, orgID, "", nil
	case entities.CredentialResourceProject:
		if resourceID == "" {
			return "", "", "", &validation.Error{Field: "resourceId", Message: "resourceId is required for project shares"}
		}
		if _, err := s.projects.FindByID(ctx, orgID, resourceID); err != nil {
			return "", "", "", err
		}
		return resourceType, resourceID, resourceID, nil
	case entities.CredentialResourceExpense:
		if resourceID == "" {
			return "", "", "", &validation.Error{Field: "resourceId", Message: "resourceId is required for expense shares"}
		}
		expense, err := s.expenses.FindByID(ctx, orgID, resourceID)
		if err != nil {
			return "", "", "", err
		}
		return resourceType, resourceID, expense.ProjectID, nil
	default:
		return "", "", "", &validation.Error{Field: "resourceType", Message: "resourceType must be org, project, or expense"}
	}
}

func scopesForResource(resourceType string) []string {
	switch resourceType {
	case entities.CredentialResourceOrg:
		return append([]string(nil), shareScopes...)
	case entities.CredentialResourceProject:
		return []string{"orgs.get", "projects.get", "expenses.list", "expenses.summary", "expenses.get"}
	case entities.CredentialResourceExpense:
		return []string{"orgs.get", "projects.get", "expenses.get"}
	default:
		return nil
	}
}

func normalizeCredentialExpiry(expiresAt *time.Time) (*time.Time, error) {
	if expiresAt == nil {
		expiresAt = timePtr(time.Now().UTC().Add(30 * 24 * time.Hour))
	}
	expiresAt = timePtr(expiresAt.UTC())
	if !expiresAt.After(time.Now().UTC()) {
		return nil, &validation.Error{Field: "expiresAt", Message: "expiresAt must be in the future"}
	}
	if expiresAt.After(time.Now().UTC().Add(maxShareLifetime)) {
		return nil, &validation.Error{Field: "expiresAt", Message: "expiresAt must be within 365 days"}
	}
	return expiresAt, nil
}

// List returns credentials matching the filter.
func (s *CredentialService) List(ctx context.Context, in ListCredentialsInput) ([]entities.Credential, error) {
	orgID := strings.TrimSpace(in.OrgID)
	if orgID == "" {
		return nil, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}

	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		return nil, &validation.Error{Field: "kind", Message: "kind is required"}
	}
	if kind != entities.CredentialKindShare && kind != entities.CredentialKindAPIKey {
		return nil, &validation.Error{Field: "kind", Message: "kind must be share or api_key"}
	}

	resourceType := strings.TrimSpace(in.ResourceType)
	if resourceType != "" {
		if kind != entities.CredentialKindShare {
			return nil, &validation.Error{Field: "resourceType", Message: "resourceType is only valid for share credentials"}
		}
		switch resourceType {
		case entities.CredentialResourceOrg, entities.CredentialResourceProject, entities.CredentialResourceExpense:
		default:
			return nil, &validation.Error{Field: "resourceType", Message: "resourceType must be org, project, or expense"}
		}
	}

	resourceIDs := make([]string, 0, len(in.ResourceIDs))
	for _, id := range in.ResourceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		resourceIDs = append(resourceIDs, id)
	}
	if len(resourceIDs) > maxCredentialIDs {
		return nil, &validation.Error{Field: "resourceIds", Message: "resourceIds must contain at most 100 ids"}
	}
	if len(resourceIDs) > 0 && kind != entities.CredentialKindShare {
		return nil, &validation.Error{Field: "resourceIds", Message: "resourceIds is only valid for share credentials"}
	}

	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	items, err := s.store.List(ctx, dao.CredentialFilter{
		OrgID:        orgID,
		Kind:         kind,
		ResourceType: resourceType,
		ResourceIDs:  resourceIDs,
		Active:       in.Active,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	return items, nil
}

// Revoke revokes a share.
func (s *CredentialService) Revoke(ctx context.Context, orgID, id string) error {
	orgID = strings.TrimSpace(orgID)
	id = strings.TrimSpace(id)
	if orgID == "" {
		return &validation.Error{Field: "orgId", Message: "orgId is required"}
	}
	if id == "" {
		return &validation.Error{Field: "shareId", Message: "shareId is required"}
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := s.store.Revoke(ctx, orgID, id, time.Now().UTC()); err != nil {
		return fmt.Errorf("revoke credential: %w", err)
	}
	return nil
}

// ResolveToken returns the active credential for a raw share token.
func (s *CredentialService) ResolveToken(ctx context.Context, raw string) (entities.Credential, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, shareTokenPrefix) {
		return entities.Credential{}, entities.ErrCredentialNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	credential, err := s.store.FindByTokenHash(ctx, hashToken(raw))
	if err != nil {
		return entities.Credential{}, err
	}
	if credential.Kind != entities.CredentialKindShare || !credential.Active(time.Now().UTC()) {
		return entities.Credential{}, entities.ErrCredentialNotFound
	}
	return credential, nil
}

func newShareToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate share token: %w", err)
	}
	token := shareTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func timePtr(t time.Time) *time.Time {
	return &t
}
