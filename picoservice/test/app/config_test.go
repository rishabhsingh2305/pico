package apptest

import (
	"strings"
	"testing"

	"picoservice/internal/app"
)

func TestConfigFromEnv(t *testing.T) {
	validSecret := strings.Repeat("s", 32)

	tests := []struct {
		name     string
		secret   string
		baseURL  string
		from     string
		smtpAddr string
		username string
		password string
		wantErr  bool
	}{
		{
			name:     "valid without auth",
			secret:   validSecret,
			baseURL:  "http://localhost:8080/",
			from:     "dev@example.com",
			smtpAddr: "localhost:1025",
		},
		{
			name:     "valid with auth",
			secret:   validSecret,
			baseURL:  "https://api.example.com",
			from:     "noreply@example.com",
			smtpAddr: "smtp.example.com:587",
			username: "user",
			password: "password",
		},
		{
			name:     "short secret",
			secret:   "short",
			baseURL:  "https://api.example.com",
			from:     "noreply@example.com",
			smtpAddr: "smtp.example.com:587",
			wantErr:  true,
		},
		{
			name:     "invalid base url",
			secret:   validSecret,
			baseURL:  "not-a-url",
			from:     "noreply@example.com",
			smtpAddr: "smtp.example.com:587",
			wantErr:  true,
		},
		{
			name:     "base url with query",
			secret:   validSecret,
			baseURL:  "https://api.example.com?x=1",
			from:     "noreply@example.com",
			smtpAddr: "smtp.example.com:587",
			wantErr:  true,
		},
		{
			name:     "missing from",
			secret:   validSecret,
			baseURL:  "https://api.example.com",
			smtpAddr: "smtp.example.com:587",
			wantErr:  true,
		},
		{
			name:     "invalid from",
			secret:   validSecret,
			baseURL:  "https://api.example.com",
			from:     "not-an-email",
			smtpAddr: "smtp.example.com:587",
			wantErr:  true,
		},
		{
			name:    "missing smtp address",
			secret:  validSecret,
			baseURL: "https://api.example.com",
			from:    "noreply@example.com",
			wantErr: true,
		},
		{
			name:     "bad smtp address",
			secret:   validSecret,
			baseURL:  "https://api.example.com",
			from:     "noreply@example.com",
			smtpAddr: "smtp.example.com",
			wantErr:  true,
		},
		{
			name:     "username without password",
			secret:   validSecret,
			baseURL:  "https://api.example.com",
			from:     "noreply@example.com",
			smtpAddr: "smtp.example.com:587",
			username: "user",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.secret)
			t.Setenv("PUBLIC_BASE_URL", tt.baseURL)
			t.Setenv("EMAIL_FROM", tt.from)
			t.Setenv("SMTP_ADDR", tt.smtpAddr)
			t.Setenv("SMTP_USERNAME", tt.username)
			t.Setenv("SMTP_PASSWORD", tt.password)

			cfg, err := app.ConfigFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := cfg.PublicBaseURL, strings.TrimRight(tt.baseURL, "/"); got != want {
				t.Fatalf("expected base URL %q, got %q", want, got)
			}
		})
	}
}
