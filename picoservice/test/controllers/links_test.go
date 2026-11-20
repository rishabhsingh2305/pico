package controllerstest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestLinkCreate(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	service := &fakes.LinkService{
		CreateFunc: func(_ context.Context, in services.CreateLinkInput) (entities.Link, error) {
			if in.OrgID != "org-1" || in.UserID != "user-1" {
				t.Fatalf("unexpected identity: %q %q", in.OrgID, in.UserID)
			}
			if in.Slug != "launch" || in.URL != "https://example.com" {
				t.Fatalf("unexpected input: %+v", in)
			}
			return entities.Link{
				ID:        "link-1",
				OrgID:     in.OrgID,
				UserID:    in.UserID,
				Slug:      in.Slug,
				URL:       in.URL,
				Title:     in.Title,
				Tags:      in.Tags,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	var purged string
	purger := &fakes.LinkPurger{PurgeFunc: func(_ context.Context, url string) error {
		purged = url
		return nil
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/links?orgId=org-1", strings.NewReader(`{"url":"https://example.com","slug":"launch","title":"Launch","tags":["marketing"]}`))
	req.Host = "api.example.com"
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewLinkController(testLogger(), service, "https://link.uppixl.com", purger).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/orgs/org-1/links/link-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["shortPath"], "/s/launch"; got != want {
		t.Fatalf("expected shortPath %q, got %v", want, got)
	}
	if got, want := body["shortUrl"], "https://link.uppixl.com/s/launch"; got != want {
		t.Fatalf("expected shortUrl %q, got %v", want, got)
	}
	if got, want := body["disabled"], false; got != want {
		t.Fatalf("expected disabled %v, got %v", want, got)
	}
	if got, want := purged, "https://link.uppixl.com/s/launch"; got != want {
		t.Fatalf("expected purge URL %q, got %q", want, got)
	}
}

func TestLinkCreateSlugConflict(t *testing.T) {
	service := &fakes.LinkService{
		CreateFunc: func(context.Context, services.CreateLinkInput) (entities.Link, error) {
			return entities.Link{}, fmt.Errorf("create link: %w", entities.ErrSlugTaken)
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/links?orgId=org-1", strings.NewReader(`{"url":"https://example.com","slug":"launch"}`))
	req = withIdentity(req, "org-1", "user-1")

	newLinkController(service).Create(rec, req)

	assertJSONError(t, rec, http.StatusConflict, "slug already in use")
}

func TestLinkCreateValidation(t *testing.T) {
	service := &fakes.LinkService{
		CreateFunc: func(context.Context, services.CreateLinkInput) (entities.Link, error) {
			return entities.Link{}, &validation.Error{Field: "url", Message: "url is required"}
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/links?orgId=org-1", strings.NewReader(`{"url":""}`))
	req = withIdentity(req, "org-1", "user-1")

	newLinkController(service).Create(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "url is required")
}

func TestLinkList(t *testing.T) {
	next := 20
	service := &fakes.LinkService{
		ListFunc: func(_ context.Context, in services.ListLinksInput) (services.LinkList, error) {
			if in.OrgID != "org-1" || in.Query != "lau" {
				t.Fatalf("unexpected input: %+v", in)
			}
			if len(in.Tags) != 2 || in.Tags[0] != "food" || in.Tags[1] != "travel" {
				t.Fatalf("unexpected tags: %v", in.Tags)
			}
			if in.Limit != 20 || in.Offset != 0 {
				t.Fatalf("unexpected pagination: %d %d", in.Limit, in.Offset)
			}
			return services.LinkList{
				Items: []entities.Link{{
					ID:        "link-1",
					OrgID:     "org-1",
					Slug:      "launch",
					URL:       "https://example.com",
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}},
				NextOffset: &next,
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/links?orgId=org-1&q=lau&tags=food,travel&limit=20", nil)
	req = withIdentity(req, "org-1", "user-1")

	newLinkController(service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body struct {
		Items      []map[string]any `json:"items"`
		NextOffset *int             `json:"nextOffset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0]["slug"] != "launch" {
		t.Fatalf("unexpected items: %v", body.Items)
	}
	if got, want := body.Items[0]["shortUrl"], "http://example.com/s/launch"; got != want {
		t.Fatalf("expected shortUrl %q, got %v", want, got)
	}
	if body.NextOffset == nil || *body.NextOffset != 20 {
		t.Fatalf("expected nextOffset 20, got %v", body.NextOffset)
	}
}

func TestLinkGetNotFound(t *testing.T) {
	service := &fakes.LinkService{
		GetFunc: func(context.Context, string, string) (entities.Link, error) {
			return entities.Link{}, fmt.Errorf("get link: %w", entities.ErrLinkNotFound)
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/links/link-1?orgId=org-1", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"linkID": "link-1"})

	newLinkController(service).Get(rec, req)

	assertJSONError(t, rec, http.StatusNotFound, "link not found")
}

func TestLinkUpdate(t *testing.T) {
	service := &fakes.LinkService{
		UpdateFunc: func(_ context.Context, in services.UpdateLinkInput) (entities.Link, error) {
			if in.OrgID != "org-1" || in.ID != "link-1" {
				t.Fatalf("unexpected scope: %+v", in)
			}
			if in.URL == nil || *in.URL != "https://example.com/new" {
				t.Fatalf("unexpected url: %v", in.URL)
			}
			if in.Disabled == nil || !*in.Disabled {
				t.Fatalf("unexpected disabled: %v", in.Disabled)
			}
			return entities.Link{
				ID:         in.ID,
				OrgID:      in.OrgID,
				Slug:       "launch",
				URL:        *in.URL,
				DisabledAt: ptrTime(time.Now().UTC()),
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			}, nil
		},
	}

	var purged string
	purger := &fakes.LinkPurger{PurgeFunc: func(_ context.Context, url string) error {
		purged = url
		return nil
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/orgs/org-1/links/link-1?orgId=org-1", strings.NewReader(`{"url":"https://example.com/new","disabled":true}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"linkID": "link-1"})

	controllers.NewLinkController(testLogger(), service, "https://link.uppixl.com", purger).Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["disabled"], true; got != want {
		t.Fatalf("expected disabled %v, got %v", want, got)
	}
	if got, want := purged, "https://link.uppixl.com/s/launch"; got != want {
		t.Fatalf("expected purge URL %q, got %q", want, got)
	}
}

func TestLinkDelete(t *testing.T) {
	service := &fakes.LinkService{
		DeleteFunc: func(_ context.Context, orgID, id string) (entities.Link, error) {
			if orgID != "org-1" || id != "link-1" {
				t.Fatalf("unexpected scope: %q %q", orgID, id)
			}
			return entities.Link{ID: id, OrgID: orgID, Slug: "launch"}, nil
		},
	}

	var purged string
	purger := &fakes.LinkPurger{PurgeFunc: func(_ context.Context, url string) error {
		purged = url
		return nil
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/orgs/org-1/links/link-1?orgId=org-1", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"linkID": "link-1"})

	controllers.NewLinkController(testLogger(), service, "https://link.uppixl.com", purger).Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body, got %q", rec.Body.String())
	}
	if got, want := purged, "https://link.uppixl.com/s/launch"; got != want {
		t.Fatalf("expected purge URL %q, got %q", want, got)
	}
}

func TestLinkPurgeFailureDoesNotFailMutation(t *testing.T) {
	service := &fakes.LinkService{
		CreateFunc: func(_ context.Context, in services.CreateLinkInput) (entities.Link, error) {
			return entities.Link{ID: "link-1", OrgID: in.OrgID, Slug: "launch"}, nil
		},
	}
	purger := &fakes.LinkPurger{PurgeFunc: func(context.Context, string) error {
		return errors.New("bunny unavailable")
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/links?orgId=org-1", strings.NewReader(`{"url":"https://example.com","slug":"launch"}`))
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewLinkController(testLogger(), service, "https://link.uppixl.com", purger).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

func TestLinkRedirect(t *testing.T) {
	var gotSlug string
	service := &fakes.LinkService{
		ResolveFunc: func(_ context.Context, slug string) (entities.Link, error) {
			gotSlug = slug
			return entities.Link{Slug: slug, URL: "https://example.com/page?q=1"}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/s/launch", nil)
	req = withRouteParams(req, map[string]string{"slug": "launch"})

	newLinkController(service).Redirect(rec, req)

	if gotSlug != "launch" {
		t.Fatalf("expected slug launch, got %q", gotSlug)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "https://example.com/page?q=1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
	if got, want := rec.Header().Get("Cache-Control"), "public, max-age=0, s-maxage=600"; got != want {
		t.Fatalf("expected Cache-Control %q, got %q", want, got)
	}
	if got, want := rec.Header().Get("X-Robots-Tag"), "noindex"; got != want {
		t.Fatalf("expected X-Robots-Tag %q, got %q", want, got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body, got %q", rec.Body.String())
	}
}

func TestLinkRedirectNotFound(t *testing.T) {
	service := &fakes.LinkService{
		ResolveFunc: func(context.Context, string) (entities.Link, error) {
			return entities.Link{}, fmt.Errorf("resolve link: %w", entities.ErrLinkNotFound)
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/s/missing", nil)
	req = withRouteParams(req, map[string]string{"slug": "missing"})

	newLinkController(service).Redirect(rec, req)

	if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Fatalf("expected Cache-Control %q, got %q", want, got)
	}
	assertJSONError(t, rec, http.StatusNotFound, "link not found")
}

func TestLinkRedirectInternalError(t *testing.T) {
	service := &fakes.LinkService{
		ResolveFunc: func(context.Context, string) (entities.Link, error) {
			return entities.Link{}, errors.New("database down")
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/s/launch", nil)
	req = withRouteParams(req, map[string]string{"slug": "launch"})

	newLinkController(service).Redirect(rec, req)

	if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Fatalf("expected Cache-Control %q, got %q", want, got)
	}
	assertJSONError(t, rec, http.StatusInternalServerError, "internal server error")
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func newLinkController(service controllers.LinkService) *controllers.LinkController {
	return controllers.NewLinkController(testLogger(), service, "", nil)
}
