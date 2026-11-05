package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

const (
	emailMaxLength    = 254
	minPasswordLength = 8
	maxPasswordLength = 72
)

// CreateUserInput contains the fields needed to register a user.
type CreateUserInput struct {
	Name     string
	Email    string
	Password string
}

// UpdateUserInput contains optional profile fields. ActorID must match UserID.
type UpdateUserInput struct {
	ActorID string
	UserID  string
	Name    *string
	Email   *string
}

// ChangePasswordInput describes a self-service password change.
type ChangePasswordInput struct {
	ActorID         string
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// ListUsersInput contains pagination fields for listing users.
type ListUsersInput struct {
	Limit  int
	Offset int
}

// EmailVerifier sends and validates email verification links.
type EmailVerifier interface {
	Send(ctx context.Context, user entities.User) error
	Verify(ctx context.Context, token string) (entities.User, error)
	Resend(ctx context.Context, email string) error
}

// UserService applies user validation and business rules.
type UserService struct {
	store        dao.UserDAO
	hasher       Hasher
	verification EmailVerifier
}

// NewUserService returns a user service backed by store, hasher, and
// verification.
func NewUserService(store dao.UserDAO, hasher Hasher, verification EmailVerifier) *UserService {
	if hasher == nil {
		hasher = NewBcryptHasher()
	}
	return &UserService{store: store, hasher: hasher, verification: verification}
}

// Create validates and registers a user.
func (s *UserService) Create(ctx context.Context, in CreateUserInput) (entities.User, error) {
	name, err := validation.Name(in.Name)
	if err != nil {
		return entities.User{}, err
	}
	email, err := validateEmail(in.Email)
	if err != nil {
		return entities.User{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return entities.User{}, err
	}

	passwordHash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return entities.User{}, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := entities.User{
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}

	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	created, err := s.store.Insert(opCtx, user, passwordHash)
	cancel()
	if err != nil {
		return entities.User{}, fmt.Errorf("create user: %w", err)
	}

	if err := s.verification.Send(ctx, created); err != nil {
		// Use a fresh context so cleanup still has a chance when the request
		// context has been canceled or its deadline has expired.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cleanupCancel()

		cleanupErr := s.store.Delete(cleanupCtx, created.ID)
		if cleanupErr != nil {
			return entities.User{}, fmt.Errorf("send verification email: %w", errors.Join(err, fmt.Errorf("cleanup user: %w", cleanupErr)))
		}
		return entities.User{}, fmt.Errorf("send verification email: %w", err)
	}
	return created, nil
}

// Get returns a user by ID.
func (s *UserService) Get(ctx context.Context, id string) (entities.User, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return entities.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// List returns a page of users.
func (s *UserService) List(ctx context.Context, in ListUsersInput) ([]entities.User, error) {
	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	items, err := s.store.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return items, nil
}

// Update changes a user's name and/or email. Users may only update themselves.
func (s *UserService) Update(ctx context.Context, in UpdateUserInput) (entities.User, error) {
	if in.ActorID != in.UserID {
		return entities.User{}, entities.ErrForbidden
	}
	if in.Name == nil && in.Email == nil {
		return entities.User{}, &validation.Error{Field: "body", Message: "no fields to update"}
	}

	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	current, err := s.store.FindByID(opCtx, in.UserID)
	if err != nil {
		return entities.User{}, fmt.Errorf("get user: %w", err)
	}

	name := current.Name
	if in.Name != nil {
		name, err = validation.Name(*in.Name)
		if err != nil {
			return entities.User{}, err
		}
	}

	email := current.Email
	if in.Email != nil {
		email, err = validateEmail(*in.Email)
		if err != nil {
			return entities.User{}, err
		}
	}

	emailChanged := email != current.Email
	emailVerified := current.EmailVerified && !emailChanged
	updated, err := s.store.Update(opCtx, in.UserID, name, email, emailVerified, time.Now().UTC())
	if err != nil {
		return entities.User{}, fmt.Errorf("update user: %w", err)
	}

	if emailChanged {
		if err := s.verification.Send(ctx, updated); err != nil {
			return entities.User{}, fmt.Errorf("send verification email: %w", err)
		}
	}
	return updated, nil
}

// VerifyEmail validates a verification token and marks the user's email as
// verified. It returns the verified user.
func (s *UserService) VerifyEmail(ctx context.Context, token string) (entities.User, error) {
	return s.verification.Verify(ctx, token)
}

// ResendVerification sends a new verification email when the address is
// eligible.
func (s *UserService) ResendVerification(ctx context.Context, email string) error {
	return s.verification.Resend(ctx, email)
}

// ChangePassword verifies the current password and stores a new hash. Users
// may only change their own password.
func (s *UserService) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	if in.ActorID != in.UserID {
		return entities.ErrForbidden
	}
	if err := validatePassword(in.NewPassword); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	currentHash, err := s.store.PasswordHash(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("get password: %w", err)
	}
	if err := s.hasher.Compare(currentHash, in.CurrentPassword); err != nil {
		if errors.Is(err, entities.ErrPasswordMismatch) {
			return entities.ErrInvalidPassword
		}
		return fmt.Errorf("verify password: %w", err)
	}

	newHash, err := s.hasher.Hash(in.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.store.UpdatePassword(ctx, in.UserID, newHash, time.Now().UTC()); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// Exists reports whether a user exists. It is used by identity middleware.
func (s *UserService) Exists(ctx context.Context, id string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	exists, err := s.store.Exists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check user: %w", err)
	}
	return exists, nil
}

func validateEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", &validation.Error{Field: "email", Message: "email is required"}
	}
	if len(email) > emailMaxLength {
		return "", &validation.Error{Field: "email", Message: "email must be at most 254 characters"}
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, "@") {
		return "", &validation.Error{Field: "email", Message: "email must be a valid email address"}
	}
	return email, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return &validation.Error{Field: "password", Message: "password must be at least 8 characters"}
	}
	if len(password) > maxPasswordLength {
		return &validation.Error{Field: "password", Message: "password must be at most 72 bytes"}
	}
	return nil
}
