package routestest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestExpenseShareAccess(t *testing.T) {
	fakeServices := fakes.NewServices()
	fakeServices.Credentials.ResolveFunc = func(context.Context, string) (entities.Credential, error) {
		return entities.Credential{
			ID:           "cred-1",
			Kind:         entities.CredentialKindShare,
			OrgID:        "org-1",
			ResourceType: entities.CredentialResourceExpense,
			ResourceID:   "expense-1",
			ProjectID:    "project-1",
			Scopes:       []string{"orgs.get", "projects.get", "expenses.get"},
		}, nil
	}
	fakeServices.Expenses.GetFunc = func(_ context.Context, orgID, id string) (entities.Expense, error) {
		return entities.Expense{ID: id, OrgID: orgID, ProjectID: "project-1"}, nil
	}
	r := newRouter(t, fakeServices, stubPinger{})

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "own expense", method: http.MethodGet, path: "/expenses/expense-1?orgId=org-1", wantStatus: http.StatusOK},
		{name: "other expense", method: http.MethodGet, path: "/expenses/expense-2?orgId=org-1", wantStatus: http.StatusNotFound},
		{name: "list denied", method: http.MethodGet, path: "/expenses?orgId=org-1", wantStatus: http.StatusForbidden},
		{name: "create denied", method: http.MethodPost, path: "/expenses?orgId=org-1", body: `{"projectId":"project-1","amount":100,"spentAt":"2026-01-02T03:04:05Z"}`, wantStatus: http.StatusForbidden},
		{name: "other org denied", method: http.MethodGet, path: "/expenses/expense-1?orgId=org-2", wantStatus: http.StatusForbidden},
		{name: "org denied", method: http.MethodGet, path: "/orgs/org-1?orgId=org-1", wantStatus: http.StatusOK},
		{name: "user route denied", method: http.MethodGet, path: "/users/me", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Share share-token")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestProjectShareFiltersExpenses(t *testing.T) {
	fakeServices := fakes.NewServices()
	fakeServices.Credentials.ResolveFunc = func(context.Context, string) (entities.Credential, error) {
		return entities.Credential{
			ID:           "cred-1",
			Kind:         entities.CredentialKindShare,
			OrgID:        "org-1",
			ResourceType: entities.CredentialResourceProject,
			ResourceID:   "project-1",
			ProjectID:    "project-1",
			Scopes:       []string{"orgs.get", "projects.get", "expenses.list", "expenses.summary", "expenses.get"},
		}, nil
	}
	var gotProjectIDs []string
	fakeServices.Expenses.ListFunc = func(_ context.Context, in services.ListExpensesInput) (services.ExpenseList, error) {
		gotProjectIDs = in.ProjectIDs
		return services.ExpenseList{Items: []entities.Expense{}}, nil
	}
	fakeServices.Expenses.SummarizeFunc = func(_ context.Context, in services.SummarizeExpensesInput) ([]services.ExpenseSummaryGroup, error) {
		gotProjectIDs = in.ProjectIDs
		return []services.ExpenseSummaryGroup{}, nil
	}
	fakeServices.Expenses.GetFunc = func(_ context.Context, orgID, id string) (entities.Expense, error) {
		projectID := "project-1"
		if id == "expense-2" {
			projectID = "project-2"
		}
		return entities.Expense{ID: id, OrgID: orgID, ProjectID: projectID}, nil
	}
	r := newRouter(t, fakeServices, stubPinger{})

	for _, path := range []string{
		"/expenses?orgId=org-1&projectIds=project-2",
		"/expenses/summary?orgId=org-1&projectIds=project-2",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Share share-token")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", path, rec.Code, rec.Body.String())
		}
		if len(gotProjectIDs) != 1 || gotProjectIDs[0] != "project-1" {
			t.Fatalf("%s: expected project-1 filter, got %v", path, gotProjectIDs)
		}
	}

	for path, want := range map[string]int{
		"/expenses/expense-1?orgId=org-1": http.StatusOK,
		"/expenses/expense-2?orgId=org-1": http.StatusNotFound,
		"/projects/project-2?orgId=org-1": http.StatusNotFound,
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Share share-token")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s: expected %d, got %d: %s", path, want, rec.Code, rec.Body.String())
		}
	}
}
