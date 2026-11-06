package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"picoservice/internal/entities"
)

const (
	accessTokenIssuer   = "picoservice"
	accessTokenAudience = "picoservice-api"
	accessTokenPurpose  = "access"
	accessTokenTTL      = 7 * 24 * time.Hour
)

// dummyPasswordHash is compared when the email is unknown so login timing does
// not reveal whether an account exists.
const dummyPasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// AuthUserStore provides the user operations login and session need.
type AuthUserStore interface {
	FindByEmail(ctx context.Context, email string) (entities.User, error)
	FindByID(ctx context.Context, id string) (entities.User, error)
	PasswordHash(ctx context.Context, id string) (string, error)
}

// AccessClaims contains the access token claims.
type AccessClaims struct {
	Type string `json:"type"`
	jwt.RegisteredClaims
}

// LoginResult contains the identity and token issued by a successful login.
type LoginResult struct {
	User      entities.User
	Token     string
	ExpiresAt time.Time
}

// AuthService validates credentials and issues stateless access tokens.
type AuthService struct {
	users  AuthUserStore
	hasher Hasher
	secret []byte
}

// NewAuthService returns an auth service that signs tokens with secret.
func NewAuthService(users AuthUserStore, hasher Hasher, secret string) (*AuthService, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 bytes")
	}
	if hasher == nil {
		hasher = NewBcryptHasher()
	}
	return &AuthService{users: users, hasher: hasher, secret: []byte(secret)}, nil
}

// Login verifies an email and password and issues an access token. Unknown
// emails and wrong passwords share one error to avoid account enumeration.
func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	normalized, err := validateEmail(email)
	if err != nil || password == "" {
		return LoginResult{}, entities.ErrInvalidCredentials
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	user, err := s.users.FindByEmail(ctx, normalized)
	if errors.Is(err, entities.ErrUserNotFound) {
		_ = s.hasher.Compare(dummyPasswordHash, password)
		return LoginResult{}, entities.ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("find user: %w", err)
	}

	hash, err := s.users.PasswordHash(ctx, user.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("get password: %w", err)
	}
	if err := s.hasher.Compare(hash, password); err != nil {
		if errors.Is(err, entities.ErrPasswordMismatch) {
			return LoginResult{}, entities.ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("verify password: %w", err)
	}

	if !user.EmailVerified {
		return LoginResult{}, entities.ErrEmailNotVerified
	}

	token, expiresAt, err := s.issue(user, time.Now().UTC())
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Token: token, ExpiresAt: expiresAt}, nil
}

// Session validates token and returns the current user and token expiry.
func (s *AuthService) Session(ctx context.Context, token string) (entities.User, time.Time, error) {
	claims, err := s.Parse(token)
	if err != nil {
		return entities.User{}, time.Time{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	user, err := s.users.FindByID(ctx, claims.Subject)
	if errors.Is(err, entities.ErrUserNotFound) {
		return entities.User{}, time.Time{}, entities.ErrInvalidToken
	}
	if err != nil {
		return entities.User{}, time.Time{}, fmt.Errorf("get user: %w", err)
	}
	return user, claims.ExpiresAt.Time, nil
}

// ParseSubject validates token and returns its subject. It performs no
// database work and is used by identity middleware.
func (s *AuthService) ParseSubject(token string) (string, error) {
	claims, err := s.Parse(token)
	if err != nil {
		return "", err
	}
	return claims.Subject, nil
}

// Parse validates raw and returns its access claims.
func (s *AuthService) Parse(raw string) (AccessClaims, error) {
	claims := AccessClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		&claims,
		func(*jwt.Token) (any, error) {
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(accessTokenIssuer),
		jwt.WithAudience(accessTokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return AccessClaims{}, entities.ErrInvalidToken
	}
	if claims.Type != accessTokenPurpose || claims.Subject == "" {
		return AccessClaims{}, entities.ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) issue(user entities.User, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(accessTokenTTL)
	claims := AccessClaims{
		Type: accessTokenPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    accessTokenIssuer,
			Audience:  jwt.ClaimStrings{accessTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}
