package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// VerificationOTP contains the stored state for an email verification code.
type VerificationOTP struct {
	UserID    string
	Hash      string
	ExpiresAt time.Time
	Attempts  int
}

// UserDAO persists users. Password hashes are passed separately so they never
// become part of the public entity.
type UserDAO interface {
	Insert(ctx context.Context, user entities.User, passwordHash string) (entities.User, error)
	FindByID(ctx context.Context, id string) (entities.User, error)
	FindByEmail(ctx context.Context, email string) (entities.User, error)
	PasswordHash(ctx context.Context, id string) (string, error)
	List(ctx context.Context, limit, offset int) ([]entities.User, error)
	Update(ctx context.Context, id, name, email string, emailVerified bool, updatedAt time.Time) (entities.User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error
	MarkEmailVerified(ctx context.Context, id, email string, verifiedAt time.Time) (entities.User, error)
	SetVerificationSentAt(ctx context.Context, id string, sentAt time.Time) error
	SetVerificationOTP(ctx context.Context, id, hash string, expiresAt time.Time) error
	VerificationOTP(ctx context.Context, email string) (VerificationOTP, error)
	IncrementVerificationOTPAttempts(ctx context.Context, id string) (int, error)
	ClearVerificationOTP(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}
