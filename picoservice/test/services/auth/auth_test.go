package authtest

import (
	"context"
	"errors"
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
	if result.ExpiresAt.Before(time.Now().Add(6 * 24 * time.Hour)) {
		t.Fatalf("expected a long-lived token, got expiry %v", result.ExpiresAt)
	}

	subject, err := service.ParseSubject(result.Token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if got, want := subject, "user-1"; got != want {
		t.Fatalf("expected subject %q, got %q", want, got)
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
	if _, err := services.NewAuthService(newFakeStore(), fakeHasher{}, "short"); err == nil {
		t.Fatal("expected an error for a short secret")
	}
}

func newService(t *testing.T, store *fakeStore) *services.AuthService {
	t.Helper()
	service, err := services.NewAuthService(store, fakeHasher{}, testSecret)
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
