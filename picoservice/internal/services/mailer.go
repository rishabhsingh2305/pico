package services

import "context"

// Mailer sends transactional email.
type Mailer interface {
	SendVerification(ctx context.Context, to, name, verifyURL string) error
}
