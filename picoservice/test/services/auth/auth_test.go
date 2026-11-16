package authtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"picoservice/internal/entities"
	"picoservice/internal/services"
)

const testSecret = "01234567890123456789012345678901"

func TestLoginIssuesToken(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	result, err := service.Login(context.Background(), " ADA@EXAMPLE.COM ", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected an access token")
	}
	if got, want := result.User.ID, "user-1"; got != want {
		t.Fatalf("expected user %q, got %q", want, got)
	}
	if result.ExpiresAt.Before(time.Now().Add(20 * time.Minute)) {
		t.Fatalf("expected a short-lived access token, got expiry %v", result.ExpiresAt)
	}
	if result.RefreshExpiresAt.Before(time.Now().Add(29 * 24 * time.Hour)) {
		t.Fatalf("expected a long-lived refresh token, got expiry %v", result.RefreshExpiresAt)
	}
	if result.RefreshToken == "" {
		t.Fatal("expected a refresh token")
	}

	subject, _, err := service.ParseAccess(result.Token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if got, want := subject, "user-1"; got != want {
		t.Fatalf("expected subject %q, got %q", want, got)
	}
}

func TestLoginIncludesOrgRoles(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service, err := services.NewAuthService(store, fakeHasher{}, testSecret, fakeRoles{
		roles: map[string][]string{
			"org-owned":  {entities.RoleWildcard},
			"org-family": {"expense.viewer", "project.viewer"},
		},
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	result, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	subject, orgs, err := service.ParseAccess(result.Token)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if got, want := subject, "user-1"; got != want {
		t.Fatalf("expected subject %q, got %q", want, got)
	}
	assertRoleList(t, orgs["org-owned"], []string{entities.RoleWildcard})
	assertRoleList(t, orgs["org-family"], []string{"expense.viewer", "project.viewer"})
}

func TestRefreshReissuesRoles(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	roles := &fakeRoles{roles: map[string][]string{"org-1": {"expense.viewer"}}}
	service, err := services.NewAuthService(store, fakeHasher{}, testSecret, roles)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	login, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	roles.roles = map[string][]string{"org-1": {entities.RoleWildcard}}

	refreshed, err := service.Refresh(context.Background(), login.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.Token == login.Token {
		t.Fatal("expected a new access token")
	}
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("expected a rotated refresh token")
	}

	subject, orgs, err := service.ParseAccess(refreshed.Token)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if got, want := subject, "user-1"; got != want {
		t.Fatalf("expected subject %q, got %q", want, got)
	}
	assertRoleList(t, orgs["org-1"], []string{entities.RoleWildcard})
}

func TestTokenSizeForTwentyOrganizations(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	roles := make(map[string][]string, 20)
	for i := 0; i < 20; i++ {
		roles[fmt.Sprintf("org-%02d", i)] = []string{
			"org.viewer", "member.viewer", "project.editor", "expense.editor",
		}
	}
	service, err := services.NewAuthService(store, fakeHasher{}, testSecret, fakeRoles{roles: roles})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	result, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	t.Logf("access token size for 20 organizations: %d bytes", len(result.Token))
	if len(result.Token) > 6000 {
		t.Fatalf("access token is %d bytes; keep it below 6000 for header limits", len(result.Token))
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	login, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, err := service.Refresh(context.Background(), login.Token); !errors.Is(err, entities.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
	if _, err := service.Refresh(context.Background(), login.RefreshToken); err != nil {
		t.Fatalf("expected the refresh token to work, got %v", err)
	}
}

func TestRefreshTokenIsNotAccessToken(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	login, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	subject, _, err := service.ParseAccess(login.RefreshToken)
	if !errors.Is(err, entities.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
	if subject != "" {
		t.Fatalf("expected no subject, got %q", subject)
	}
}

func TestLoginFailsWhenRoleResolutionFails(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service, err := services.NewAuthService(store, fakeHasher{}, testSecret, fakeRoles{err: errors.New("database down")})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	if _, err := service.Login(context.Background(), "ada@example.com", "correct horse"); err == nil {
		t.Fatal("expected a role resolution error")
	}
}

func TestRefreshFailsWhenRoleResolutionFails(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	login, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	failing, err := services.NewAuthService(store, fakeHasher{}, testSecret, fakeRoles{err: errors.New("database down")})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	if _, err := failing.Refresh(context.Background(), login.RefreshToken); err == nil {
		t.Fatal("expected a role resolution error")
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{name: "wrong password", email: "ada@example.com", password: "wrong"},
		{name: "unknown email", email: "grace@example.com", password: "correct horse"},
		{name: "invalid email", email: "not-an-email", password: "correct horse"},
		{name: "empty password", email: "ada@example.com", password: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, entities.ErrInvalidCredentials) {
				t.Fatalf("expected ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

func TestLoginRequiresVerifiedEmail(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", false)
	service := newService(t, store)

	_, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if !errors.Is(err, entities.ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
}

func TestSession(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	result, err := service.Login(context.Background(), "ada@example.com", "correct horse")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user, expiresAt, err := service.Session(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if got, want := user.ID, "user-1"; got != want {
		t.Fatalf("expected user %q, got %q", want, got)
	}
	if expiresAt.Unix() != result.ExpiresAt.Unix() {
		t.Fatalf("expected expiry %v, got %v", result.ExpiresAt, expiresAt)
	}
}

func TestSessionRejectsInvalidTokens(t *testing.T) {
	store := newFakeStore()
	store.add("user-1", "ada@example.com", "hashed:correct horse", true)
	service := newService(t, store)

	verification, err := services.NewVerificationTokens(testSecret)
	if err != nil {
		t.Fatalf("verification tokens: %v", err)
	}
	verificationToken, err := verification.Issue(entities.User{ID: "user-1", Email: "ada@example.com"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("issue verification token: %v", err)
	}

	expired := signToken(t, time.Now().UTC().Add(-time.Hour))

	tests := []struct {
		name  string
		token string
	}{
		{name: "garbage", token: "not-a-token"},
		{name: "verification token", token: verificationToken},
		{name: "expired", token: expired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := service.Session(context.Background(), tt.token); !errors.Is(err, entities.ErrInvalidToken) {
				t.Fatalf("expected ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestNewAuthServiceRequiresSecret(t *testing.T) {
	if _, err := services.NewAuthService(newFakeStore(), fakeHasher{}, "short", fakeRoles{}); err == nil {
		t.Fatal("expected an error for a short secret")
	}
}

func newService(t *testing.T, store *fakeStore) *services.AuthService {
	t.Helper()
	service, err := services.NewAuthService(store, fakeHasher{}, testSecret, fakeRoles{})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	return service
}

func signToken(t *testing.T, expiresAt time.Time) string {
	t.Helper()
	claims := services.AccessClaims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			Issuer:    "picoservice",
			Audience:  jwt.ClaimStrings{"picoservice-api"},
			IssuedAt:  jwt.NewNumericDate(expiresAt.Add(-time.Hour)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

type fakeHasher struct{}

type fakeRoles struct {
	roles map[string][]string
	err   error
}

func (r fakeRoles) RolesForUser(context.Context, string) (map[string][]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.roles == nil {
		return map[string][]string{}, nil
	}
	return r.roles, nil
}

func assertRoleList(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected roles %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected roles %v, got %v", want, got)
		}
	}
}

func (fakeHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakeHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return entities.ErrPasswordMismatch
	}
	return nil
}

type fakeStore struct {
	users  map[string]entities.User
	hashes map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: make(map[string]entities.User), hashes: make(map[string]string)}
}

func (s *fakeStore) add(id, email, hash string, verified bool) {
	s.users[id] = entities.User{ID: id, Email: email, EmailVerified: verified}
	s.hashes[id] = hash
}

func (s *fakeStore) FindByEmail(_ context.Context, email string) (entities.User, error) {
	for _, user := range s.users {
		if strings.EqualFold(user.Email, email) {
			return user, nil
		}
	}
	return entities.User{}, entities.ErrUserNotFound
}

func (s *fakeStore) FindByID(_ context.Context, id string) (entities.User, error) {
	user, ok := s.users[id]
	if !ok {
		return entities.User{}, entities.ErrUserNotFound
	}
	return user, nil
}

func (s *fakeStore) PasswordHash(_ context.Context, id string) (string, error) {
	hash, ok := s.hashes[id]
	if !ok {
		return "", entities.ErrUserNotFound
	}
	return hash, nil
}
