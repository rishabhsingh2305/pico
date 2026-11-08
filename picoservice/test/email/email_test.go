package emailtest

import (
	"context"
	"strings"
	"testing"

	"picoservice/internal/email"
)

func TestNewSMTP(t *testing.T) {
	tests := []struct {
		name    string
		cfg     email.Config
		wantErr bool
	}{
		{
			name: "valid without auth",
			cfg: email.Config{
				Addr: "localhost:1025",
				From: "dev@example.com",
			},
		},
		{
			name: "valid with auth",
			cfg: email.Config{
				Addr:     "smtp.example.com:587",
				Username: "user",
				Password: "password",
				From:     "noreply@example.com",
			},
		},
		{
			name:    "missing address",
			cfg:     email.Config{From: "dev@example.com"},
			wantErr: true,
		},
		{
			name:    "bad port",
			cfg:     email.Config{Addr: "localhost:not-a-port", From: "dev@example.com"},
			wantErr: true,
		},
		{
			name:    "missing from",
			cfg:     email.Config{Addr: "localhost:1025"},
			wantErr: true,
		},
		{
			name:    "username without password",
			cfg:     email.Config{Addr: "localhost:1025", Username: "user", From: "dev@example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := email.NewSMTP(tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSendVerificationRejectsInvalidRecipient(t *testing.T) {
	sender, err := email.NewSMTP(email.Config{
		Addr: "localhost:1025",
		From: "dev@example.com",
	})
	if err != nil {
		t.Fatalf("create sender: %v", err)
	}

	if err := sender.SendVerification(context.Background(), "not-an-email", "Ada", "http://localhost/verify", "123456"); err == nil {
		t.Fatal("expected an invalid recipient error")
	}
}

func TestBuildVerificationMessage(t *testing.T) {
	message := string(email.BuildVerificationMessage(
		"Pico <noreply@example.com>",
		"ada@example.com",
		"<Ada>",
		"https://example.com/auth/verify-email?token=abc&next=1",
		"123456",
	))

	checks := []string{
		"From: Pico <noreply@example.com>",
		"To: ada@example.com",
		"Subject: Verify your email - Pico",
		"Content-Type: multipart/alternative",
		"Confirm your email address for Pico.",
		"Hi <Ada>,",
		"https://example.com/auth/verify-email?token=abc&next=1",
		"&lt;Ada&gt;",
		"https://example.com/auth/verify-email?token=abc&amp;next=1",
		"Verify email",
		"123456",
		"Your verification code",
		"Expires in 10 minutes",
		"This code expires in 10 minutes.",
		"background-color:#f4f1ea",
		"background-color:#fbf8f1",
		"background-color:#c63c1e",
		"Pico &middot; Shared expense tracking for small teams",
		"expires in 24 hours",
	}
	for _, check := range checks {
		if !strings.Contains(message, check) {
			t.Fatalf("expected message to contain %q, got:\n%s", check, message)
		}
	}
	if strings.Contains(message, "%!") {
		t.Fatalf("expected formatted message without errors, got:\n%s", message)
	}
}
