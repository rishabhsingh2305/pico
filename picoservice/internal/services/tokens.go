package services

import (
	"crypto/rand"
	"encoding/hex"
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
	refreshTokenPurpose = "refresh"
	accessTokenTTL      = 30 * time.Minute
	refreshTokenTTL     = 30 * 24 * time.Hour
)

// AccessClaims contains the token claims. Orgs maps an organization ID to the
// user's feature roles, or entities.RoleWildcard for an organization the user
// owns.
type AccessClaims struct {
	Type string              `json:"type"`
	Orgs map[string][]string `json:"orgs,omitempty"`
	jwt.RegisteredClaims
}

// tokenManager signs and validates access and refresh tokens. It has no user or
// membership knowledge.
type tokenManager struct {
	secret []byte
}

func newTokenManager(secret string) (*tokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 bytes")
	}
	return &tokenManager{secret: []byte(secret)}, nil
}

type tokenPair struct {
	accessToken      string
	accessExpiresAt  time.Time
	refreshToken     string
	refreshExpiresAt time.Time
}

func (m *tokenManager) issuePair(userID string, orgs map[string][]string, now time.Time) (tokenPair, error) {
	access, accessExpiresAt, err := m.sign(userID, accessTokenPurpose, orgs, now, accessTokenTTL)
	if err != nil {
		return tokenPair{}, err
	}
	refresh, refreshExpiresAt, err := m.sign(userID, refreshTokenPurpose, nil, now, refreshTokenTTL)
	if err != nil {
		return tokenPair{}, err
	}
	return tokenPair{
		accessToken:      access,
		accessExpiresAt:  accessExpiresAt,
		refreshToken:     refresh,
		refreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (m *tokenManager) parseAccess(raw string) (string, map[string][]string, error) {
	claims, err := m.parse(raw, accessTokenPurpose)
	if err != nil {
		return "", nil, err
	}
	orgs := claims.Orgs
	if orgs == nil {
		orgs = map[string][]string{}
	}
	return claims.Subject, orgs, nil
}

func (m *tokenManager) parse(raw, purpose string) (AccessClaims, error) {
	claims := AccessClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		&claims,
		func(*jwt.Token) (any, error) {
			return m.secret, nil
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
	if claims.Type != purpose || claims.Subject == "" {
		return AccessClaims{}, entities.ErrInvalidToken
	}
	return claims, nil
}

func (m *tokenManager) sign(subject, purpose string, orgs map[string][]string, now time.Time, ttl time.Duration) (string, time.Time, error) {
	tokenID, err := newTokenID()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := now.Add(ttl)
	claims := AccessClaims{
		Type: purpose,
		Orgs: orgs,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   subject,
			Issuer:    accessTokenIssuer,
			Audience:  jwt.ClaimStrings{accessTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func newTokenID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
