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
	"picoservice/internal/validation"
	"picoservice/test/fakes"
)

func TestProjectCreate(t *testing.T) {
	service := &fakes.ProjectService{
		CreateFunc: func(_ context.Context, in services.CreateProjectInput) (entities.Project, error) {
			if in.OrgID != "org-1" || in.UserID != "user-1" {
				t.Fatalf("unexpected identity: %q %q", in.OrgID, in.UserID)
			}
			if got, want := in.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("expected tags %v, got %v", want, got)
			}
			return entities.Project{
				ID:        "project-1",
				OrgID:     in.OrgID,
				UserID:    in.UserID,
				Name:      in.Name,
				Currency:  in.Currency,
				Tags:      in.Tags,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/projects?orgId=org-1&userId=user-1", strings.NewReader(`{"name":"Launch","currency":"USD","tags":["food","travel"]}`))
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewProjectController(testLogger(), service).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/projects/project-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["currency"], "USD"; got != want {
		t.Fatalf("expected currency %q, got %v", want, got)
	}
	tags, ok := body["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "food" || tags[1] != "travel" {
		t.Fatalf("expected tags [food travel], got %v", body["tags"])
	}
}

func TestProjectListTags(t *testing.T) {
	service := &fakes.ProjectService{
		ListFunc: func(_ context.Context, in services.ListProjectsInput) ([]entities.Project, error) {
			if got, want := in.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("expected tags %v, got %v", want, got)
			}
			return []entities.Project{{ID: "project-1", Tags: []string{"food"}}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/projects?orgId=org-1&userId=user-1&tags=food,travel", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewProjectController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestProjectUpdateTags(t *testing.T) {
	service := &fakes.ProjectService{
		UpdateFunc: func(_ context.Context, in services.UpdateProjectInput) (entities.Project, error) {
			if in.Name != nil {
				t.Fatalf("expected tags-only update, got name %v", *in.Name)
			}
			if in.Tags == nil || len(*in.Tags) != 2 || (*in.Tags)[0] != "food" || (*in.Tags)[1] != "travel" {
				t.Fatalf("expected tags [food travel], got %v", in.Tags)
			}
			return entities.Project{ID: in.ID, Tags: *in.Tags}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/projects/project-1?orgId=org-1&userId=user-1", strings.NewReader(`{"tags":["food","travel"]}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"projectID": "project-1"})

	controllers.NewProjectController(testLogger(), service).Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestProjectCreateValidation(t *testing.T) {
	service := &fakes.ProjectService{
		CreateFunc: func(context.Context, services.CreateProjectInput) (entities.Project, error) {
			return entities.Project{}, &validation.Error{Field: "name", Message: "name is required"}
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/projects?orgId=org-1&userId=user-1", strings.NewReader(`{"name":"","currency":"USD"}`))
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewProjectController(testLogger(), service).Create(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "name is required")
}

func TestProjectGetNotFound(t *testing.T) {
	service := &fakes.ProjectService{
		GetFunc: func(context.Context, string, string) (entities.Project, error) {
			return entities.Project{}, entities.ErrProjectNotFound
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/projects/missing?orgId=org-1&userId=user-1", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"projectID": "missing"})

	controllers.NewProjectController(testLogger(), service).Get(rec, req)

	assertJSONError(t, rec, http.StatusNotFound, "project not found")
}
