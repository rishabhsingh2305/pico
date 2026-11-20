package cdntest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"picoservice/internal/cdn"
)

func TestNewBunnyPurger(t *testing.T) {
	tests := []struct {
		name    string
		cfg     cdn.BunnyConfig
		wantErr bool
	}{
		{name: "valid", cfg: cdn.BunnyConfig{APIKey: "key"}},
		{name: "missing key", cfg: cdn.BunnyConfig{}, wantErr: true},
		{name: "invalid endpoint", cfg: cdn.BunnyConfig{APIKey: "key", Endpoint: "not-a-url"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := cdn.NewBunnyPurger(tt.cfg); (err != nil) != tt.wantErr {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestBunnyPurgerPurge(t *testing.T) {
	var gotMethod, gotURL, gotAsync, gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotURL = r.URL.Query().Get("url")
		gotAsync = r.URL.Query().Get("async")
		gotKey = r.Header.Get("AccessKey")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	purger, err := cdn.NewBunnyPurger(cdn.BunnyConfig{
		APIKey:   "secret-key",
		Endpoint: server.URL + "/purge",
		Client:   server.Client(),
	})
	if err != nil {
		t.Fatalf("create purger: %v", err)
	}

	const target = "https://link.uppixl.com/s/launch?x=1"
	if err := purger.Purge(context.Background(), target); err != nil {
		t.Fatalf("purge: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotURL != target {
		t.Fatalf("expected target %q, got %q", target, gotURL)
	}
	if gotAsync != "false" {
		t.Fatalf("expected async=false, got %q", gotAsync)
	}
	if gotKey != "secret-key" {
		t.Fatalf("expected AccessKey header, got %q", gotKey)
	}
}

func TestBunnyPurgerPurgeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	purger, err := cdn.NewBunnyPurger(cdn.BunnyConfig{
		APIKey:   "key",
		Endpoint: server.URL,
		Client:   server.Client(),
	})
	if err != nil {
		t.Fatalf("create purger: %v", err)
	}

	if err := purger.Purge(context.Background(), "https://link.uppixl.com/s/launch"); err == nil {
		t.Fatal("expected a purge error")
	}
}

func TestBunnyPurgerRejectsInvalidTarget(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	t.Cleanup(server.Close)

	purger, err := cdn.NewBunnyPurger(cdn.BunnyConfig{
		APIKey:   "key",
		Endpoint: server.URL,
		Client:   server.Client(),
	})
	if err != nil {
		t.Fatalf("create purger: %v", err)
	}

	if err := purger.Purge(context.Background(), "not-a-url"); err == nil {
		t.Fatal("expected an invalid target error")
	}
	if called {
		t.Fatal("expected no purge request for an invalid target")
	}
}
