package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

const (
	verificationIssuer   = "picoservice"
	verificationAudience = "email-verification"
	verificationPurpose  = "email_verification"
	verificationTokenTTL = 24 * time.Hour
	// ResendThrottle is the minimum interval between verification resends.
	ResendThrottle   = time.Minute
	emailSendTimeout = 5 * time.Second
)

var (
	// ErrInvalidVerificationToken is returned for malformed, expired, or
	// mismatched email verification tokens.
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
	// ErrEmailDelivery is returned when a verification email cannot be sent.
	ErrEmailDelivery = errors.New("could not send verification email")
)

// VerificationUserStore provides the user operations email verification needs.
type VerificationUserStore interface {
	FindByID(ctx context.Context, id string) (entities.User, error)
	FindByEmail(ctx context.Context, email string) (entities.User, error)
	MarkEmailVerified(ctx context.Context, id, email string, verifiedAt time.Time) (entities.User, error)
	SetVerificationSentAt(ctx context.Context, id string, sentAt time.Time) error
	SetVerificationOTP(ctx context.Context, id, hash string, expiresAt time.Time) error
	VerificationOTP(ctx context.Context, email string) (dao.VerificationOTP, error)
	IncrementVerificationOTPAttempts(ctx context.Context, id string) (int, error)
	ClearVerificationOTP(ctx context.Context, id string) error
}

// VerificationClaims contains the email verification token claims.
type VerificationClaims struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// VerificationTokens issues and validates email verification JWTs.
type VerificationTokens struct {
	secret []byte
}

// NewVerificationTokens returns a token issuer using secret.
func NewVerificationTokens(secret string) (*VerificationTokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 bytes")
	}
	return &VerificationTokens{secret: []byte(secret)}, nil
}

// Issue returns a signed verification token for user.
func (t *VerificationTokens) Issue(user entities.User, now time.Time) (string, error) {
	claims := VerificationClaims{
		Email:   user.Email,
		Purpose: verificationPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    verificationIssuer,
			Audience:  jwt.ClaimStrings{verificationAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(verificationTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign verification token: %w", err)
	}
	return signed, nil
}

// Parse validates raw and returns its verification claims.
func (t *VerificationTokens) Parse(raw string) (VerificationClaims, error) {
	claims := VerificationClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		&claims,
		func(*jwt.Token) (any, error) {
			return t.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(verificationIssuer),
		jwt.WithAudience(verificationAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return VerificationClaims{}, ErrInvalidVerificationToken
	}
	if claims.Purpose != verificationPurpose || claims.Subject == "" || claims.Email == "" {
		return VerificationClaims{}, ErrInvalidVerificationToken
	}
	return claims, nil
}

// EmailVerificationService sends and validates email verification links and
// codes.
type EmailVerificationService struct {
	users   VerificationUserStore
	tokens  *VerificationTokens
	mailer  Mailer
	baseURL string
	otp     *OTPHasher
}

// NewEmailVerificationService returns a verification service. baseURL is the
// public URL used to build verification links.
func NewEmailVerificationService(users VerificationUserStore, tokens *VerificationTokens, mailer Mailer, baseURL string) *EmailVerificationService {
	return &EmailVerificationService{
		users:   users,
		tokens:  tokens,
		mailer:  mailer,
		baseURL: strings.TrimRight(baseURL, "/"),
		otp:     newOTPHasher(tokens.secret),
	}
}

// Send generates a token and verification code and emails them to user.
func (s *EmailVerificationService) Send(ctx context.Context, user entities.User) error {
	now := time.Now().UTC()
	token, err := s.tokens.Issue(user, now)
	if err != nil {
		return fmt.Errorf("issue verification token: %w", err)
	}

	code, err := generateOTP()
	if err != nil {
		return err
	}

	otpCtx, otpCancel := context.WithTimeout(ctx, operationTimeout)
	defer otpCancel()
	if err := s.users.SetVerificationOTP(otpCtx, user.ID, s.otp.Hash(code), now.Add(otpTTL)); err != nil {
		return fmt.Errorf("store verification otp: %w", err)
	}

	verifyURL := s.baseURL + "/verify-email#token=" + url.QueryEscape(token)
	sendCtx, cancel := context.WithTimeout(ctx, emailSendTimeout)
	defer cancel()

	if err := s.mailer.SendVerification(sendCtx, user.Email, user.Name, verifyURL, code); err != nil {
		return fmt.Errorf("%w: %w", ErrEmailDelivery, err)
	}
	return nil
}

// Verify validates token and marks the matching user's email as verified. It
// returns the verified user.
func (s *EmailVerificationService) Verify(ctx context.Context, token string) (entities.User, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return entities.User{}, ErrInvalidVerificationToken
	}

	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	user, err := s.users.FindByID(opCtx, claims.Subject)
	if errors.Is(err, entities.ErrUserNotFound) {
		return entities.User{}, ErrInvalidVerificationToken
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("get user: %w", err)
	}
	if !strings.EqualFold(user.Email, claims.Email) {
		return entities.User{}, ErrInvalidVerificationToken
	}
	if user.EmailVerified {
		return user, nil
	}

	verified, err := s.users.MarkEmailVerified(opCtx, user.ID, user.Email, time.Now().UTC())
	if err != nil {
		if errors.Is(err, entities.ErrUserNotFound) {
			return entities.User{}, ErrInvalidVerificationToken
		}
		return entities.User{}, fmt.Errorf("mark email verified: %w", err)
	}
	return verified, nil
}

// VerifyOTP validates a verification code and marks the matching user's email
// as verified.
func (s *EmailVerificationService) VerifyOTP(ctx context.Context, rawEmail, code string) (entities.User, error) {
	email, err := validateEmail(rawEmail)
	if err != nil {
		return entities.User{}, entities.ErrInvalidVerificationCode
	}

	code = strings.TrimSpace(code)
	if !isOTP(code) {
		return entities.User{}, entities.ErrInvalidVerificationCode
	}

	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	user, err := s.users.FindByEmail(opCtx, email)
	if errors.Is(err, entities.ErrUserNotFound) {
		return entities.User{}, entities.ErrInvalidVerificationCode
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("find user: %w", err)
	}
	if user.EmailVerified {
		return user, nil
	}

	otp, err := s.users.VerificationOTP(opCtx, email)
	if errors.Is(err, entities.ErrUserNotFound) {
		return entities.User{}, entities.ErrInvalidVerificationCode
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("get verification otp: %w", err)
	}
	if otp.Hash == "" || otp.ExpiresAt.IsZero() || !otp.ExpiresAt.After(time.Now().UTC()) {
		return entities.User{}, entities.ErrInvalidVerificationCode
	}
	if otp.Attempts >= otpMaxAttempts {
		return entities.User{}, entities.ErrTooManyVerificationAttempts
	}

	if !s.otp.Compare(otp.Hash, code) {
		attempts, err := s.users.IncrementVerificationOTPAttempts(opCtx, user.ID)
		if err != nil {
			return entities.User{}, fmt.Errorf("increment verification attempts: %w", err)
		}
		if attempts >= otpMaxAttempts {
			return entities.User{}, entities.ErrTooManyVerificationAttempts
		}
		return entities.User{}, entities.ErrInvalidVerificationCode
	}

	verified, err := s.users.MarkEmailVerified(opCtx, user.ID, user.Email, time.Now().UTC())
	if errors.Is(err, entities.ErrUserNotFound) {
		return entities.User{}, entities.ErrInvalidVerificationCode
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("mark email verified: %w", err)
	}
	return verified, nil
}

func isOTP(code string) bool {
	if len(code) != otpLength {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Resend sends a new verification email when the user exists, is unverified,
// and is outside the resend throttle window.
func (s *EmailVerificationService) Resend(ctx context.Context, rawEmail string) error {
	email, err := validateEmail(rawEmail)
	if err != nil {
		return err
	}

	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	user, err := s.users.FindByEmail(opCtx, email)
	cancel()
	if errors.Is(err, entities.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user.EmailVerified {
		return nil
	}

	now := time.Now().UTC()
	if user.VerificationSentAt != nil && now.Sub(*user.VerificationSentAt) < ResendThrottle {
		return nil
	}

	if err := s.Send(ctx, user); err != nil {
		return err
	}

	sentCtx, sentCancel := context.WithTimeout(ctx, operationTimeout)
	defer sentCancel()
	if err := s.users.SetVerificationSentAt(sentCtx, user.ID, now); err != nil {
		return fmt.Errorf("record verification send: %w", err)
	}
	return nil
}
