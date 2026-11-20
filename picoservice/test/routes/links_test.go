package routestest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"picoservice/internal/entities"
	"picoservice/test/fakes"
)

func TestPublicShortLinkRedirect(t *testing.T) {
	fakeServices := fakes.NewServices()
	fakeServices.Links.ResolveFunc = func(_ context.Context, slug string) (entities.Link, error) {
		return entities.Link{Slug: slug, URL: "https://example.com/page"}, nil
	}
	r := newRouter(t, fakeServices, stubPinger{})

	req := httptest.NewRequest(http.MethodGet, "/s/launch", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusFound, rec.Code, rec.Body.String())
	}
	if got, want := rec.Header().Get("Location"), "https://example.com/page"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
	if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Fatalf("expected Cache-Control %q, got %q", want, got)
	}
	if got, want := rec.Header().Get("X-Robots-Tag"), "noindex"; got != want {
		t.Fatalf("expected X-Robots-Tag %q, got %q", want, got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body, got %q", rec.Body.String())
	}
}
