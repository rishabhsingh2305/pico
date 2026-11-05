package services

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"picoservice/internal/entities"
)

// Hasher hashes and verifies passwords.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// BcryptHasher implements Hasher with bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a bcrypt hasher using bcrypt's default cost.
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

// Hash returns a bcrypt hash of password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// Compare verifies password against hash. A mismatch is reported as
// entities.ErrPasswordMismatch.
func (h *BcryptHasher) Compare(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return entities.ErrPasswordMismatch
		}
		return fmt.Errorf("compare password: %w", err)
	}
	return nil
}
