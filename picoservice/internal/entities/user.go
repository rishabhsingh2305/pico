// Package entities defines the application's domain records.
package entities

import "time"

// User is a domain user record. Password hashes are deliberately excluded.
type User struct {
	ID                 string
	Name               string
	Email              string
	EmailVerified      bool
	EmailVerifiedAt    *time.Time
	VerificationSentAt *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
