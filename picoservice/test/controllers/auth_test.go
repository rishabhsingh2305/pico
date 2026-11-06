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
	"picoservice/internal/middleware"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

func TestAuthLogin(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	service := &fakes.AuthService{
		LoginFunc: func(_ context.Context, email, password string) (services.LoginResult, error) {
			if email != "ada@example.com" || password != "correct horse" {
				t.Fatalf("unexpected credentials: %q %q", email, password)
			}
			return services.LoginResult{
				User:      entities.User{ID: "user-1", Email: email, EmailVerified: true},
				Token:     "token-1",
				ExpiresAt: expiresAt,
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","password":"correct horse"}`))

	controllers.NewAuthController(testLogger(), service, false).Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	cookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, middleware.AuthCookieName) || !strings.Contains(cookie, "HttpOnly") {
		t.Fatalf("expected an HttpOnly auth cookie, got %q", cookie)
	}

	var body struct {
		User      map[string]any `json:"user"`
		ExpiresAt time.Time      `json:"expiresAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body.User["id"], "user-1"; got != want {
		t.Fatalf("expected user %q, got %v", want, got)
	}
}

func TestAuthLoginSecureCookie(t *testing.T) {
	service := &fakes.AuthService{
		LoginFunc: func(context.Context, string, string) (services.LoginResult, error) {
			return services.LoginResult{
				User:      entities.User{ID: "user-1", Email: "ada@example.com", EmailVerified: true},
				Token:     "token-1",
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","password":"correct horse"}`))

	controllers.NewAuthController(testLogger(), service, true).Login(rec, req)

	cookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "Secure") || !strings.Contains(cookie, "SameSite=None") {
		t.Fatalf("expected a Secure SameSite=None cookie, got %q", cookie)
	}
}

func TestAuthLoginErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{name: "invalid credentials", err: entities.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantError: "invalid email or password"},
		{name: "email not verified", err: entities.ErrEmailNotVerified, wantStatus: http.StatusForbidden, wantError: "email not verified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakes.AuthService{
				LoginFunc: func(context.Context, string, string) (services.LoginResult, error) {
					return services.LoginResult{}, tt.err
				},
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","password":"wrong"}`))

			controllers.NewAuthController(testLogger(), service, false).Login(rec, req)

			assertJSONError(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func TestAuthLoginInvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","unknown":true}`))

	controllers.NewAuthController(testLogger(), &fakes.AuthService{}, false).Login(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "invalid request body")
}

func TestAuthLogout(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

	controllers.NewAuthController(testLogger(), &fakes.AuthService{}, false).Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if cookie := rec.Header().Get("Set-Cookie"); !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("expected the auth cookie to be cleared, got %q", cookie)
	}
}

func TestAuthSession(t *testing.T) {
	service := &fakes.AuthService{
		SessionFunc: func(_ context.Context, token string) (entities.User, time.Time, error) {
			if token != "token-1" {
				t.Fatalf("expected token-1, got %q", token)
			}
			return entities.User{ID: "user-1", Email: "ada@example.com", EmailVerified: true}, time.Now().UTC().Add(time.Hour), nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AuthCookieName, Value: "token-1"})

	controllers.NewAuthController(testLogger(), service, false).Session(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthSessionMissingCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)

	controllers.NewAuthController(testLogger(), &fakes.AuthService{}, false).Session(rec, req)

	assertJSONError(t, rec, http.StatusUnauthorized, "authentication required")
}

func TestAuthSessionInvalidToken(t *testing.T) {
	service := &fakes.AuthService{
		SessionFunc: func(context.Context, string) (entities.User, time.Time, error) {
			return entities.User{}, time.Time{}, entities.ErrInvalidToken
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AuthCookieName, Value: "bad"})

	controllers.NewAuthController(testLogger(), service, false).Session(rec, req)

	assertJSONError(t, rec, http.StatusUnauthorized, "authentication required")
}
