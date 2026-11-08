package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

// UserService is the user behavior used by the HTTP controllers.
type UserService interface {
	Create(ctx context.Context, in services.CreateUserInput) (entities.User, error)
	List(ctx context.Context, in services.ListUsersInput) ([]entities.User, error)
	Get(ctx context.Context, id string) (entities.User, error)
	Update(ctx context.Context, in services.UpdateUserInput) (entities.User, error)
	ChangePassword(ctx context.Context, in services.ChangePasswordInput) error
	VerifyEmail(ctx context.Context, token string) (entities.User, error)
	VerifyEmailOTP(ctx context.Context, email, code string) (entities.User, error)
	ResendVerification(ctx context.Context, email string) error
}

// UserController serves user endpoints.
type UserController struct {
	logger  *slog.Logger
	service UserService
}

// NewUserController returns a user controller backed by service.
func NewUserController(logger *slog.Logger, service UserService) *UserController {
	if logger == nil {
		logger = slog.Default()
	}
	return &UserController{logger: logger, service: service}
}

// Create handles POST /users.
func (c *UserController) Create(w http.ResponseWriter, r *http.Request) {
	var request createUserRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := c.service.Create(r.Context(), services.CreateUserInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	w.Header().Set(httpx.HeaderLocation, "/users/"+user.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, userResponseFrom(user))
}

// List handles GET and HEAD /users.
func (c *UserController) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	items, err := c.service.List(r.Context(), services.ListUsersInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := userListResponse{Items: make([]userResponse, 0, len(items))}
	for _, user := range items {
		response.Items = append(response.Items, userResponseFrom(user))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Get handles GET and HEAD /users/{userID}.
func (c *UserController) Get(w http.ResponseWriter, r *http.Request) {
	user, err := c.service.Get(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, userResponseFrom(user))
}

// Update handles PATCH /users/{userID}.
func (c *UserController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request updateUserRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := c.service.Update(r.Context(), services.UpdateUserInput{
		ActorID: id.UserID,
		UserID:  chi.URLParam(r, "userID"),
		Name:    request.Name,
		Email:   request.Email,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, userResponseFrom(user))
}

// ChangePassword handles POST /users/{userID}/password.
func (c *UserController) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request changePasswordRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := c.service.ChangePassword(r.Context(), services.ChangePasswordInput{
		ActorID:         id.UserID,
		UserID:          chi.URLParam(r, "userID"),
		CurrentPassword: request.CurrentPassword,
		NewPassword:     request.NewPassword,
	}); err != nil {
		c.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// VerifyEmail handles POST /auth/verify-email.
func (c *UserController) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var request verifyEmailRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	token := strings.TrimSpace(request.Token)
	if token == "" {
		httpx.WriteError(w, r, http.StatusBadRequest, "token is required")
		return
	}

	user, err := c.service.VerifyEmail(r.Context(), token)
	if err != nil {
		if errors.Is(err, services.ErrInvalidVerificationToken) {
			httpx.WriteError(w, r, http.StatusBadRequest, "invalid or expired verification token")
			return
		}
		logInternalError(c.logger, w, r, "verify email failed", err)
		return
	}

	httpx.WriteJSON(w, r, http.StatusOK, verifyEmailResponse{
		Status: "verified",
		UserID: user.ID,
		Email:  user.Email,
	})
}

// VerifyEmailOTP handles POST /auth/verify-email/otp.
func (c *UserController) VerifyEmailOTP(w http.ResponseWriter, r *http.Request) {
	var request verifyEmailOTPRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := c.service.VerifyEmailOTP(r.Context(), request.Email, request.Code)
	if err != nil {
		switch {
		case errors.Is(err, entities.ErrInvalidVerificationCode):
			httpx.WriteError(w, r, http.StatusBadRequest, "invalid or expired verification code")
		case errors.Is(err, entities.ErrTooManyVerificationAttempts):
			httpx.WriteError(w, r, http.StatusBadRequest, "too many verification attempts")
		default:
			logInternalError(c.logger, w, r, "verify email otp failed", err)
		}
		return
	}

	httpx.WriteJSON(w, r, http.StatusOK, verifyEmailResponse{
		Status: "verified",
		UserID: user.ID,
		Email:  user.Email,
	})
}

// ResendVerification handles POST /auth/verify-email/resend. It always
// returns 202 for accepted requests so callers cannot probe email addresses.
func (c *UserController) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var request resendVerificationRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := c.service.ResendVerification(r.Context(), request.Email); err != nil {
		var validationErr *validation.Error
		if errors.As(err, &validationErr) {
			httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
			return
		}
		logError(c.logger, r, "resend verification failed", err)
	}

	w.Header().Set("Retry-After", strconv.Itoa(int(services.ResendThrottle.Seconds())))
	httpx.WriteJSON(w, r, http.StatusAccepted, resendVerificationResponse{
		Status:            "sent",
		RetryAfterSeconds: int(services.ResendThrottle.Seconds()),
	})
}

func (c *UserController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrUserNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "user not found")
	case errors.Is(err, entities.ErrEmailTaken):
		httpx.WriteError(w, r, http.StatusConflict, "email already exists")
	case errors.Is(err, entities.ErrForbidden):
		httpx.WriteError(w, r, http.StatusForbidden, "forbidden")
	case errors.Is(err, entities.ErrInvalidPassword):
		httpx.WriteError(w, r, http.StatusBadRequest, "current password is incorrect")
	case errors.Is(err, services.ErrEmailDelivery):
		logError(c.logger, r, "send verification email failed", err)
		httpx.WriteError(w, r, http.StatusBadGateway, "could not send verification email")
	default:
		logInternalError(c.logger, w, r, "user request failed", err)
	}
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type verifyEmailOTPRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type resendVerificationResponse struct {
	Status            string `json:"status"`
	RetryAfterSeconds int    `json:"retryAfterSeconds"`
}

type verifyEmailResponse struct {
	Status string `json:"status"`
	UserID string `json:"userId"`
	Email  string `json:"email"`
}

type userResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type userListResponse struct {
	Items []userResponse `json:"items"`
}

func userResponseFrom(user entities.User) userResponse {
	return userResponse{
		ID:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
}
