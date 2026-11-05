package mongodbtest

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"picoservice/internal/dao/mongodb"
)

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		database string
		wantErr  string
	}{
		{
			name:     "configured",
			uri:      "mongodb://localhost:27017",
			database: "picoservice",
		},
		{
			name:     "missing uri",
			database: "picoservice",
			wantErr:  "MONGODB_URI is not set",
		},
		{
			name:    "missing database",
			uri:     "mongodb://localhost:27017",
			wantErr: "MONGODB_DATABASE is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MONGODB_URI", tt.uri)
			t.Setenv("MONGODB_DATABASE", tt.database)

			cfg, err := mongodb.ConfigFromEnv()

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := cfg.URI, tt.uri; got != want {
				t.Fatalf("expected URI %q, got %q", want, got)
			}
			if got, want := cfg.Database, tt.database; got != want {
				t.Fatalf("expected database %q, got %q", want, got)
			}
		})
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  mongodb.Config
	}{
		{
			name: "missing uri",
			cfg:  mongodb.Config{Database: "picoservice"},
		},
		{
			name: "missing database",
			cfg:  mongodb.Config{URI: "mongodb://localhost:27017"},
		},
		{
			name: "invalid uri",
			cfg:  mongodb.Config{URI: "not-a-uri", Database: "picoservice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := mongodb.New(tt.cfg); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestPingCanceledContext(t *testing.T) {
	client, err := mongodb.New(mongodb.Config{
		URI:      "mongodb://localhost:27017",
		Database: "picoservice",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	defer closeClient(t, client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := client.Ping(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPingIntegration(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to run MongoDB integration tests")
	}

	database := os.Getenv("MONGODB_TEST_DATABASE")
	if database == "" {
		database = "picoservice_test"
	}

	client, err := mongodb.New(mongodb.Config{URI: uri, Database: database})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	defer closeClient(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func closeClient(t *testing.T, client *mongodb.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Close(ctx); err != nil {
		t.Fatalf("close client: %v", err)
	}
}
