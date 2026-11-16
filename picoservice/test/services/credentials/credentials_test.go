package credentialstest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
)

func TestCreateOrgShare(t *testing.T) {
	service := newService()
	result, err := service.Create(context.Background(), services.CreateCredentialInput{
		ActorID:      "user-1",
		OrgID:        "org-1",
		ResourceType: entities.CredentialResourceOrg,
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	if !strings.HasPrefix(result.Token, "pico_share_") {
		t.Fatalf("expected share token prefix, got %q", result.Token)
	}
	if result.Credential.TokenHash == result.Token {
		t.Fatal("raw token must not be stored")
	}
	if got, want := result.Credential.TokenHash, hash(result.Token); got != want {
		t.Fatalf("expected token hash %q, got %q", want, got)
	}
	if result.Credential.ResourceType != entities.CredentialResourceOrg {
		t.Fatalf("expected org resource, got %q", result.Credential.ResourceType)
	}
	if result.Credential.ProjectID != "" {
		t.Fatalf("expected no project, got %q", result.Credential.ProjectID)
	}
	if !contains(result.Credential.Scopes, "expenses.list") {
		t.Fatalf("expected expenses.list scope, got %v", result.Credential.Scopes)
	}
	if result.Credential.ExpiresAt == nil || result.Credential.ExpiresAt.Before(time.Now().Add(29*24*time.Hour)) {
		t.Fatalf("expected roughly 30 day expiry, got %v", result.Credential.ExpiresAt)
	}
}

func TestCreateProjectShare(t *testing.T) {
	service := newService()
	result, err := service.Create(context.Background(), services.CreateCredentialInput{
		ActorID:      "user-1",
		OrgID:        "org-1",
		ResourceType: entities.CredentialResourceProject,
		ResourceID:   "project-1",
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	if result.Credential.ProjectID != "project-1" {
		t.Fatalf("expected project-1, got %q", result.Credential.ProjectID)
	}
	if contains(result.Credential.Scopes, "projects.list") {
		t.Fatal("project share must not list all projects")
	}
	if !contains(result.Credential.Scopes, "expenses.list") {
		t.Fatal("project share must list project expenses")
	}
}

func TestCreateExpenseShare(t *testing.T) {
	service := newService()
	result, err := service.Create(context.Background(), services.CreateCredentialInput{
		ActorID:      "user-1",
		OrgID:        "org-1",
		ResourceType: entities.CredentialResourceExpense,
		ResourceID:   "expense-1",
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	if result.Credential.ProjectID != "project-1" {
		t.Fatalf("expected derived project-1, got %q", result.Credential.ProjectID)
	}
	if !contains(result.Credential.Scopes, "expenses.get") {
		t.Fatal("expected expenses.get")
	}
	if contains(result.Credential.Scopes, "expenses.list") {
		t.Fatal("expense share must not list expenses")
	}
}

func TestCreateRejectsInvalidResource(t *testing.T) {
	service := newService()
	_, err := service.Create(context.Background(), services.CreateCredentialInput{
		OrgID:        "org-1",
		ResourceType: "unknown",
	})
	if err == nil {
		t.Fatal("expected a validation error")
	}

	_, err = service.Create(context.Background(), services.CreateCredentialInput{
		OrgID:        "org-1",
		ResourceType: entities.CredentialResourceExpense,
		ResourceID:   "missing",
	})
	if !errors.Is(err, entities.ErrExpenseNotFound) {
		t.Fatalf("expected ErrExpenseNotFound, got %v", err)
	}
}

func TestResolveToken(t *testing.T) {
	store := newFakeStore()
	service := services.NewCredentialService(store, fakeOrgs{}, fakeProjects{}, fakeExpenses{})
	result, err := service.Create(context.Background(), services.CreateCredentialInput{
		OrgID:        "org-1",
		ResourceType: entities.CredentialResourceProject,
		ResourceID:   "project-1",
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}

	credential, err := service.ResolveToken(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("resolve token: %v", err)
	}
	if credential.ID != result.Credential.ID {
		t.Fatalf("expected %q, got %q", result.Credential.ID, credential.ID)
	}

	if _, err := service.ResolveToken(context.Background(), "not-a-share"); !errors.Is(err, entities.ErrCredentialNotFound) {
		t.Fatalf("expected ErrCredentialNotFound, got %v", err)
	}

	expired := time.Now().UTC().Add(-time.Minute)
	credential = store.items[result.Credential.ID]
	credential.ExpiresAt = &expired
	store.items[result.Credential.ID] = credential
	if _, err := service.ResolveToken(context.Background(), result.Token); !errors.Is(err, entities.ErrCredentialNotFound) {
		t.Fatalf("expected expired token to fail, got %v", err)
	}

	credential = store.items[result.Credential.ID]
	credential.ExpiresAt = nil
	revoked := time.Now().UTC()
	credential.RevokedAt = &revoked
	store.items[result.Credential.ID] = credential
	if _, err := service.ResolveToken(context.Background(), result.Token); !errors.Is(err, entities.ErrCredentialNotFound) {
		t.Fatalf("expected revoked token to fail, got %v", err)
	}
}

func TestRevoke(t *testing.T) {
	store := newFakeStore()
	service := services.NewCredentialService(store, fakeOrgs{}, fakeProjects{}, fakeExpenses{})
	result, err := service.Create(context.Background(), services.CreateCredentialInput{
		OrgID:        "org-1",
		ResourceType: entities.CredentialResourceOrg,
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}

	if err := service.Revoke(context.Background(), "org-1", result.Credential.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if store.items[result.Credential.ID].RevokedAt == nil {
		t.Fatal("expected revokedAt to be set")
	}
}

func TestListFilters(t *testing.T) {
	store := newFakeStore()
	service := services.NewCredentialService(store, fakeOrgs{}, fakeProjects{}, fakeExpenses{})
	for _, resourceType := range []string{entities.CredentialResourceOrg, entities.CredentialResourceProject} {
		resourceID := ""
		if resourceType == entities.CredentialResourceProject {
			resourceID = "project-1"
		}
		_, err := service.Create(context.Background(), services.CreateCredentialInput{
			OrgID:        "org-1",
			ResourceType: resourceType,
			ResourceID:   resourceID,
		})
		if err != nil {
			t.Fatalf("create %s: %v", resourceType, err)
		}
	}

	items, err := service.List(context.Background(), services.ListCredentialsInput{
		OrgID:        "org-1",
		Kind:         entities.CredentialKindShare,
		ResourceType: entities.CredentialResourceProject,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one project share, got %d", len(items))
	}
}

func newService() *services.CredentialService {
	return services.NewCredentialService(newFakeStore(), fakeOrgs{}, fakeProjects{}, fakeExpenses{})
}

func hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type fakeStore struct {
	items map[string]entities.Credential
	next  int
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: make(map[string]entities.Credential)}
}

func (s *fakeStore) Insert(_ context.Context, credential entities.Credential) (entities.Credential, error) {
	s.next++
	credential.ID = "cred-" + string(rune('0'+s.next))
	s.items[credential.ID] = credential
	return credential, nil
}

func (s *fakeStore) FindByID(_ context.Context, orgID, id string) (entities.Credential, error) {
	credential, ok := s.items[id]
	if !ok || credential.OrgID != orgID {
		return entities.Credential{}, entities.ErrCredentialNotFound
	}
	return credential, nil
}

func (s *fakeStore) FindByTokenHash(_ context.Context, tokenHash string) (entities.Credential, error) {
	for _, credential := range s.items {
		if credential.TokenHash == tokenHash {
			return credential, nil
		}
	}
	return entities.Credential{}, entities.ErrCredentialNotFound
}

func (s *fakeStore) List(_ context.Context, filter dao.CredentialFilter) ([]entities.Credential, error) {
	items := make([]entities.Credential, 0)
	for _, credential := range s.items {
		if credential.OrgID != filter.OrgID {
			continue
		}
		if filter.Kind != "" && credential.Kind != filter.Kind {
			continue
		}
		if filter.ResourceType != "" && credential.ResourceType != filter.ResourceType {
			continue
		}
		if len(filter.ResourceIDs) > 0 && !hasID(filter.ResourceIDs, credential.ResourceID) {
			continue
		}
		if filter.Active && !credential.Active(time.Now().UTC()) {
			continue
		}
		items = append(items, credential)
	}
	return items, nil
}

func (s *fakeStore) Revoke(_ context.Context, orgID, id string, revokedAt time.Time) error {
	credential, ok := s.items[id]
	if !ok || credential.OrgID != orgID {
		return entities.ErrCredentialNotFound
	}
	credential.RevokedAt = &revokedAt
	s.items[id] = credential
	return nil
}

func hasID(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

type fakeOrgs struct{}

func (fakeOrgs) FindByID(_ context.Context, id string) (entities.Org, error) {
	if id == "org-1" {
		return entities.Org{ID: id}, nil
	}
	return entities.Org{}, entities.ErrOrgNotFound
}

type fakeProjects struct{}

func (fakeProjects) FindByID(_ context.Context, orgID, id string) (entities.Project, error) {
	if orgID == "org-1" && id == "project-1" {
		return entities.Project{ID: id, OrgID: orgID}, nil
	}
	return entities.Project{}, entities.ErrProjectNotFound
}

type fakeExpenses struct{}

func (fakeExpenses) FindByID(_ context.Context, orgID, id string) (entities.Expense, error) {
	if orgID == "org-1" && id == "expense-1" {
		return entities.Expense{ID: id, OrgID: orgID, ProjectID: "project-1"}, nil
	}
	return entities.Expense{}, entities.ErrExpenseNotFound
}
