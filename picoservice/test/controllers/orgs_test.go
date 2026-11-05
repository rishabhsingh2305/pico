package controllerstest

import (
	"context"
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

func TestOrgCreate(t *testing.T) {
	service := &fakes.OrgService{
		CreateFunc: func(_ context.Context, in services.CreateOrgInput) (entities.Org, error) {
			if got, want := in.ActorID, "user-1"; got != want {
				t.Fatalf("expected actor %q, got %q", want, got)
			}
			if got, want := len(in.Tags), 2; got != want {
				t.Fatalf("expected %d tags, got %d", want, got)
			}
			if got, want := in.Tags[0], "team"; got != want {
				t.Fatalf("expected first tag %q, got %q", want, got)
			}
			return entities.Org{
				ID:        "org-1",
				Name:      in.Name,
				UserID:    in.ActorID,
				Tags:      in.Tags,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs?userId=user-1", strings.NewReader(`{"name":"Acme","tags":["team","finance"]}`))
	req = withIdentity(req, "", "user-1")

	controllers.NewOrgController(testLogger(), service).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/orgs/org-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
	if !strings.Contains(rec.Body.String(), `"tags":["team","finance"]`) {
		t.Fatalf("expected tags in response, got %q", rec.Body.String())
	}
}

func TestOrgUpdateForbidden(t *testing.T) {
	service := &fakes.OrgService{
		UpdateFunc: func(context.Context, services.UpdateOrgInput) (entities.Org, error) {
			return entities.Org{}, entities.ErrForbidden
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orgs/org-1", strings.NewReader(`{"name":"New"}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewOrgController(testLogger(), service).Update(rec, req)

	assertJSONError(t, rec, http.StatusForbidden, "forbidden")
}

func TestOrgUpdateTags(t *testing.T) {
	service := &fakes.OrgService{
		UpdateFunc: func(_ context.Context, in services.UpdateOrgInput) (entities.Org, error) {
			if in.Name != nil {
				t.Fatalf("expected name to be omitted, got %q", *in.Name)
			}
			if in.Tags == nil || len(*in.Tags) != 1 || (*in.Tags)[0] != "team" {
				t.Fatalf("expected tags [team], got %v", in.Tags)
			}
			return entities.Org{ID: in.OrgID, Name: "Acme", Tags: *in.Tags}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orgs/org-1", strings.NewReader(`{"tags":["team"]}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"orgID": "org-1"})

	controllers.NewOrgController(testLogger(), service).Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"tags":["team"]`) {
		t.Fatalf("expected tags in response, got %q", rec.Body.String())
	}
}

func TestOrgGetNotFound(t *testing.T) {
	service := &fakes.OrgService{
		GetFunc: func(context.Context, string) (entities.Org, error) {
			return entities.Org{}, entities.ErrOrgNotFound
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/missing", nil)
	req = withRouteParams(req, map[string]string{"orgID": "missing"})

	controllers.NewOrgController(testLogger(), service).Get(rec, req)

	assertJSONError(t, rec, http.StatusNotFound, "org not found")
}

func TestOrgList(t *testing.T) {
	service := &fakes.OrgService{
		ListFunc: func(_ context.Context, in services.ListOrgsInput) ([]entities.Org, error) {
			if got, want := len(in.Tags), 2; got != want {
				t.Fatalf("expected %d filter tags, got %d", want, got)
			}
			if got, want := in.Tags[0], "team"; got != want {
				t.Fatalf("expected first filter tag %q, got %q", want, got)
			}
			return []entities.Org{{ID: "org-1", Name: "Acme", UserID: "user-1", Tags: []string{"finance", "team"}}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs?userId=user-1&tags=team,finance", nil)
	req = withIdentity(req, "", "user-1")

	controllers.NewOrgController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":"org-1"`) {
		t.Fatalf("expected org in response, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"tags":["finance","team"]`) {
		t.Fatalf("expected tags in response, got %q", rec.Body.String())
	}
}
