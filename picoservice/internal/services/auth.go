package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"picoservice/internal/entities"
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

// OrgRoleSource resolves the organization feature roles written into a token.
type OrgRoleSource interface {
	RolesForUser(ctx context.Context, userID string) (map[string][]string, error)
}

// LoginResult contains the identity and tokens issued by login or refresh.
type LoginResult struct {
	User             entities.User
	Token            string
	ExpiresAt        time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// AuthService validates credentials and orchestrates token issuance.
type AuthService struct {
	users  AuthUserStore
	hasher Hasher
	roles  OrgRoleSource
	tokens *tokenManager
}

// NewAuthService returns an auth service that issues tokens with secret and
// resolves organization roles through roles.
func NewAuthService(users AuthUserStore, hasher Hasher, secret string, roles OrgRoleSource) (*AuthService, error) {
	tokens, err := newTokenManager(secret)
	if err != nil {
		return nil, err
	}
	if hasher == nil {
		hasher = NewBcryptHasher()
	}
	return &AuthService{users: users, hasher: hasher, roles: roles, tokens: tokens}, nil
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

	return s.issueForUser(ctx, user)
}

// Refresh exchanges a valid refresh token for a new access token and a rotated
// refresh token. It re-queries the user's memberships so role changes take
// effect without waiting for the access token to expire.
func (s *AuthService) Refresh(ctx context.Context, raw string) (LoginResult, error) {
	claims, err := s.tokens.parse(raw, refreshTokenPurpose)
	if err != nil {
		return LoginResult{}, entities.ErrInvalidToken
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	user, err := s.users.FindByID(ctx, claims.Subject)
	if errors.Is(err, entities.ErrUserNotFound) {
		return LoginResult{}, entities.ErrInvalidToken
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("get user: %w", err)
	}
	if !user.EmailVerified {
		return LoginResult{}, entities.ErrInvalidToken
	}

	return s.issueForUser(ctx, user)
}

func (s *AuthService) issueForUser(ctx context.Context, user entities.User) (LoginResult, error) {
	orgs, err := s.orgRoles(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}

	pair, err := s.tokens.issuePair(user.ID, orgs, time.Now().UTC())
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		User:             user,
		Token:            pair.accessToken,
		ExpiresAt:        pair.accessExpiresAt,
		RefreshToken:     pair.refreshToken,
		RefreshExpiresAt: pair.refreshExpiresAt,
	}, nil
}

func (s *AuthService) orgRoles(ctx context.Context, userID string) (map[string][]string, error) {
	if s.roles == nil {
		return map[string][]string{}, nil
	}
	roles, err := s.roles.RolesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve org roles: %w", err)
	}
	return roles, nil
}

// Session validates token and returns the current user and token expiry.
func (s *AuthService) Session(ctx context.Context, token string) (entities.User, time.Time, error) {
	claims, err := s.tokens.parse(token, accessTokenPurpose)
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

// ParseAccess validates token and returns its subject and organization role
// map without database work.
func (s *AuthService) ParseAccess(token string) (string, map[string][]string, error) {
	return s.tokens.parseAccess(token)
}
