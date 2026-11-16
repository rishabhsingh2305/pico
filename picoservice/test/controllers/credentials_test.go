package controllerstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"picoservice/internal/controllers"
	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestShareCreate(t *testing.T) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	service := &fakes.CredentialService{
		CreateFunc: func(_ context.Context, in services.CreateCredentialInput) (services.CreateCredentialResult, error) {
			if in.OrgID != "org-1" || in.ResourceType != entities.CredentialResourceProject || in.ResourceID != "project-1" {
				t.Fatalf("unexpected create input: %+v", in)
			}
			return services.CreateCredentialResult{
				Credential: entities.Credential{
					ID:           "cred-1",
					Kind:         entities.CredentialKindShare,
					OrgID:        "org-1",
					ResourceType: entities.CredentialResourceProject,
					ResourceID:   "project-1",
					ProjectID:    "project-1",
					Scopes:       []string{"orgs.get", "projects.get"},
					TokenHash:    "hash",
					CreatedAt:    time.Now().UTC(),
					ExpiresAt:    &expiresAt,
				},
				Token: "pico_share_raw",
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/shares", strings.NewReader(`{"resourceType":"project","resourceId":"project-1"}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewCredentialController(testLogger(), service).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/orgs/org-1/shares/cred-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := body["token"], "pico_share_raw"; got != want {
		t.Fatalf("expected token %q, got %v", want, got)
	}
	if got, want := body["sharePath"], "/shared?orgId=org-1&projectId=project-1&resourceId=project-1&resourceType=project#share=pico_share_raw"; got != want {
		t.Fatalf("expected share path %q, got %v", want, got)
	}
	if _, ok := body["tokenHash"]; ok {
		t.Fatal("response must not include the token hash")
	}
}

func TestShareListDoesNotLeakToken(t *testing.T) {
	service := &fakes.CredentialService{
		ListFunc: func(context.Context, services.ListCredentialsInput) ([]entities.Credential, error) {
			return []entities.Credential{{
				ID:           "cred-1",
				Kind:         entities.CredentialKindShare,
				OrgID:        "org-1",
				ResourceType: entities.CredentialResourceProject,
				ResourceID:   "project-1",
				Scopes:       []string{"orgs.get"},
				TokenHash:    "hash",
				CreatedAt:    time.Now().UTC(),
			}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/shares", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewCredentialController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "token") {
		t.Fatalf("list response must not contain a token: %s", rec.Body.String())
	}
}

func TestShareStatusGroupsActiveLinks(t *testing.T) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	service := &fakes.CredentialService{
		ListFunc: func(_ context.Context, in services.ListCredentialsInput) ([]entities.Credential, error) {
			if !in.Active || in.ResourceType != entities.CredentialResourceExpense {
				t.Fatalf("unexpected status filter: %+v", in)
			}
			return []entities.Credential{
				{ID: "cred-1", ResourceID: "expense-1", ResourceType: entities.CredentialResourceExpense, ExpiresAt: &expiresAt},
				{ID: "cred-2", ResourceID: "expense-1", ResourceType: entities.CredentialResourceExpense, ExpiresAt: &expiresAt},
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/shares/status?resourceType=expense&resourceIds=expense-1,expense-2", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewCredentialController(testLogger(), service).Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []struct {
			ResourceID  string `json:"resourceId"`
			ActiveCount int    `json:"activeCount"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ResourceID != "expense-1" || body.Items[0].ActiveCount != 2 {
		t.Fatalf("unexpected status response: %+v", body.Items)
	}
}

func TestShareRevoke(t *testing.T) {
	service := &fakes.CredentialService{
		RevokeFunc: func(_ context.Context, orgID, id string) error {
			if orgID != "org-1" || id != "cred-1" {
				t.Fatalf("unexpected revoke input: %q %q", orgID, id)
			}
			return nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/orgs/org-1/shares/cred-1", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1", "shareID": "cred-1"})

	controllers.NewCredentialController(testLogger(), service).Revoke(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
