package controllerstest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/middleware"
	"picoservice/internal/services"
	"picoservice/internal/validation"
	"picoservice/test/fakes"
)

func TestUserCreate(t *testing.T) {
	service := &fakes.UserService{
		CreateFunc: func(_ context.Context, in services.CreateUserInput) (entities.User, error) {
			if got, want := in.Name, "Ada"; got != want {
				t.Fatalf("expected name %q, got %q", want, got)
			}
			return entities.User{
				ID:        "user-1",
				Name:      in.Name,
				Email:     in.Email,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"correct horse"}`))

	controllers.NewUserController(testLogger(), service).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/users/user-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
	if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "hash") {
		t.Fatalf("response leaked password fields: %q", rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["email"], "ada@example.com"; got != want {
		t.Fatalf("expected email %q, got %v", want, got)
	}
	if got, ok := body["emailVerified"]; !ok || got != false {
		t.Fatalf("expected emailVerified false, got %v", got)
	}
}

func TestUserCreateConflict(t *testing.T) {
	service := &fakes.UserService{
		CreateFunc: func(context.Context, services.CreateUserInput) (entities.User, error) {
			return entities.User{}, entities.ErrEmailTaken
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"correct horse"}`))

	controllers.NewUserController(testLogger(), service).Create(rec, req)

	assertJSONError(t, rec, http.StatusConflict, "email already exists")
}

func TestUserCreateInvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Ada","unknown":true}`))

	controllers.NewUserController(testLogger(), &fakes.UserService{}).Create(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "invalid request body")
}

func TestUserGetNotFound(t *testing.T) {
	service := &fakes.UserService{
		GetFunc: func(context.Context, string) (entities.User, error) {
			return entities.User{}, entities.ErrUserNotFound
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/missing", nil)

	controllers.NewUserController(testLogger(), service).Get(rec, req)

	assertJSONError(t, rec, http.StatusNotFound, "user not found")
}

func TestUserUpdateForbidden(t *testing.T) {
	service := &fakes.UserService{
		UpdateFunc: func(context.Context, services.UpdateUserInput) (entities.User, error) {
			return entities.User{}, entities.ErrForbidden
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/user-1", strings.NewReader(`{"name":"Grace"}`))
	req = withIdentity(req, "", "user-2")
	req = withRouteParams(req, map[string]string{"userID": "user-1"})

	controllers.NewUserController(testLogger(), service).Update(rec, req)

	assertJSONError(t, rec, http.StatusForbidden, "forbidden")
}

func TestUserChangePassword(t *testing.T) {
	t.Run("wrong current password", func(t *testing.T) {
		service := &fakes.UserService{
			ChangePasswordFunc: func(context.Context, services.ChangePasswordInput) error {
				return entities.ErrInvalidPassword
			},
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users/user-1/password", strings.NewReader(`{"currentPassword":"wrong","newPassword":"new password"}`))
		req = withIdentity(req, "", "user-1")
		req = withRouteParams(req, map[string]string{"userID": "user-1"})

		controllers.NewUserController(testLogger(), service).ChangePassword(rec, req)

		assertJSONError(t, rec, http.StatusBadRequest, "current password is incorrect")
	})

	t.Run("success", func(t *testing.T) {
		service := &fakes.UserService{
			ChangePasswordFunc: func(_ context.Context, in services.ChangePasswordInput) error {
				if got, want := in.UserID, "user-1"; got != want {
					t.Fatalf("expected user %q, got %q", want, got)
				}
				return nil
			},
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users/user-1/password", strings.NewReader(`{"currentPassword":"old password","newPassword":"new password"}`))
		req = withIdentity(req, "", "user-1")
		req = withRouteParams(req, map[string]string{"userID": "user-1"})

		controllers.NewUserController(testLogger(), service).ChangePassword(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
	})
}

func TestUserList(t *testing.T) {
	service := &fakes.UserService{
		ListFunc: func(context.Context, services.ListUsersInput) ([]entities.User, error) {
			return []entities.User{{ID: "user-1", Name: "Ada", Email: "ada@example.com"}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users?userId=user-1", nil)
	req = withIdentity(req, "", "user-1")

	controllers.NewUserController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":"user-1"`) {
		t.Fatalf("expected user in response, got %q", rec.Body.String())
	}
}

func withIdentity(r *http.Request, orgID, userID string) *http.Request {
	return r.WithContext(middleware.WithContext(r.Context(), middleware.Identity{OrgID: orgID, UserID: userID}))
}

func withRouteParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestUserCreateEmailFailure(t *testing.T) {
	service := &fakes.UserService{
		CreateFunc: func(context.Context, services.CreateUserInput) (entities.User, error) {
			return entities.User{}, services.ErrEmailDelivery
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"correct horse"}`))

	controllers.NewUserController(testLogger(), service).Create(rec, req)

	assertJSONError(t, rec, http.StatusBadGateway, "could not send verification email")
}

func TestUserVerifyEmail(t *testing.T) {
	service := &fakes.UserService{
		VerifyEmailFunc: func(_ context.Context, token string) (entities.User, error) {
			if got, want := token, "token-1"; got != want {
				t.Fatalf("expected token %q, got %q", want, got)
			}
			return entities.User{ID: "user-1", Email: "ada@example.com"}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":"token-1"}`))

	controllers.NewUserController(testLogger(), service).VerifyEmail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["status"], "verified"; got != want {
		t.Fatalf("expected status %q, got %q", want, got)
	}
	if got, want := body["userId"], "user-1"; got != want {
		t.Fatalf("expected userId %q, got %q", want, got)
	}
	if got, want := body["email"], "ada@example.com"; got != want {
		t.Fatalf("expected email %q, got %q", want, got)
	}
}

func TestUserVerifyEmailMissingToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{}`))

	controllers.NewUserController(testLogger(), &fakes.UserService{}).VerifyEmail(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "token is required")
}

func TestUserVerifyEmailInvalidToken(t *testing.T) {
	service := &fakes.UserService{
		VerifyEmailFunc: func(context.Context, string) (entities.User, error) {
			return entities.User{}, services.ErrInvalidVerificationToken
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":"bad"}`))

	controllers.NewUserController(testLogger(), service).VerifyEmail(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "invalid or expired verification token")
}

func TestUserVerifyEmailInternalError(t *testing.T) {
	service := &fakes.UserService{
		VerifyEmailFunc: func(context.Context, string) (entities.User, error) {
			return entities.User{}, errors.New("database down")
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":"token-1"}`))

	controllers.NewUserController(testLogger(), service).VerifyEmail(rec, req)

	assertJSONError(t, rec, http.StatusInternalServerError, "internal server error")
}

func TestUserResendVerification(t *testing.T) {
	service := &fakes.UserService{
		ResendVerificationFunc: func(_ context.Context, email string) error {
			if got, want := email, "ada@example.com"; got != want {
				t.Fatalf("expected email %q, got %q", want, got)
			}
			return nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/resend", strings.NewReader(`{"email":"ada@example.com"}`))

	controllers.NewUserController(testLogger(), service).ResendVerification(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["status"], "sent"; got != want {
		t.Fatalf("expected status %q, got %q", want, got)
	}
}

func TestUserResendVerificationValidation(t *testing.T) {
	service := &fakes.UserService{
		ResendVerificationFunc: func(context.Context, string) error {
			return &validation.Error{Field: "email", Message: "email is required"}
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/resend", strings.NewReader(`{"email":""}`))

	controllers.NewUserController(testLogger(), service).ResendVerification(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "email is required")
}

func TestUserResendVerificationHidesInternalErrors(t *testing.T) {
	service := &fakes.UserService{
		ResendVerificationFunc: func(context.Context, string) error {
			return errors.New("database down")
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/resend", strings.NewReader(`{"email":"ada@example.com"}`))

	controllers.NewUserController(testLogger(), service).ResendVerification(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if strings.Contains(rec.Body.String(), "database") {
		t.Fatalf("response leaked an internal error: %q", rec.Body.String())
	}
}
