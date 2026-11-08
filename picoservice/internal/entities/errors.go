package entities

import "errors"

var (
	// ErrUserNotFound is returned when a user does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailTaken is returned when an email is already registered.
	ErrEmailTaken = errors.New("email already exists")
	// ErrForbidden is returned when an actor is not allowed to perform an
	// action.
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidPassword is returned when the current password is incorrect.
	ErrInvalidPassword = errors.New("current password is incorrect")
	// ErrInvalidCredentials is returned when a login email or password is
	// incorrect. It is intentionally generic to avoid account enumeration.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrEmailNotVerified is returned when a user tries to log in before
	// verifying their email address.
	ErrEmailNotVerified = errors.New("email not verified")
	// ErrInvalidToken is returned for malformed, expired, or invalid access
	// tokens.
	ErrInvalidToken = errors.New("invalid or expired token")
	// ErrInvalidVerificationCode is returned for a wrong, expired, or missing
	// email verification code.
	ErrInvalidVerificationCode = errors.New("invalid or expired verification code")
	// ErrTooManyVerificationAttempts is returned after too many failed code
	// attempts.
	ErrTooManyVerificationAttempts = errors.New("too many verification attempts")
	// ErrPasswordMismatch is returned by a Hasher when a password does not
	// match a stored hash.
	ErrPasswordMismatch = errors.New("password does not match")
	// ErrOrgNotFound is returned when an organization does not exist.
	ErrOrgNotFound = errors.New("org not found")
	// ErrMembershipNotFound is returned when a membership does not exist.
	ErrMembershipNotFound = errors.New("membership not found")
	// ErrAlreadyMember is returned when the user already belongs to the org.
	ErrAlreadyMember = errors.New("user is already a member")
	// ErrCannotRemoveOwner is returned when the owner's membership is removed.
	ErrCannotRemoveOwner = errors.New("cannot remove org owner")
	// ErrProjectNotFound is returned when a project does not exist in the
	// caller's organization.
	ErrProjectNotFound = errors.New("project not found")
	// ErrExpenseNotFound is returned when an expense does not exist in the
	// caller's organization.
	ErrExpenseNotFound = errors.New("expense not found")
)
