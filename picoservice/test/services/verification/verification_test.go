package verificationtest

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

func TestVerificationTokensRoundTrip(t *testing.T) {
	tokens := newTokens(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	now := time.Now().UTC()

	raw, err := tokens.Issue(user, now)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	claims, err := tokens.Parse(raw)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if got, want := claims.Subject, user.ID; got != want {
		t.Fatalf("expected subject %q, got %q", want, got)
	}
	if got, want := claims.Email, user.Email; got != want {
		t.Fatalf("expected email %q, got %q", want, got)
	}
	if got, want := claims.Purpose, "email_verification"; got != want {
		t.Fatalf("expected purpose %q, got %q", want, got)
	}
}

func TestVerificationTokensRejectExpired(t *testing.T) {
	tokens := newTokens(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com"}

	raw, err := tokens.Issue(user, time.Now().UTC().Add(-25*time.Hour))
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	if _, err := tokens.Parse(raw); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerificationTokensRejectWrongSecret(t *testing.T) {
	tokens := newTokens(t)
	other := newTokensWithSecret(t, strings.Repeat("t", 32))
	user := entities.User{ID: "user-1", Email: "ada@example.com"}

	raw, err := tokens.Issue(user, time.Now().UTC())
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	if _, err := other.Parse(raw); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerificationTokensRejectMalformed(t *testing.T) {
	tokens := newTokens(t)

	if _, err := tokens.Parse("not-a-token"); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestNewVerificationTokensRejectsShortSecret(t *testing.T) {
	if _, err := services.NewVerificationTokens("short"); err == nil {
		t.Fatal("expected an error for a short secret")
	}
}

func TestSend(t *testing.T) {
	service, store, mailer, tokens := newService(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	store.users[user.ID] = user

	if err := service.Send(context.Background(), user); err != nil {
		t.Fatalf("send verification: %v", err)
	}
	if mailer.calls != 1 {
		t.Fatalf("expected one email, got %d", mailer.calls)
	}
	if got, want := mailer.to, user.Email; got != want {
		t.Fatalf("expected recipient %q, got %q", want, got)
	}

	verifyURL, err := url.Parse(mailer.verifyURL)
	if err != nil {
		t.Fatalf("parse verification url: %v", err)
	}
	if got, want := verifyURL.Path, "/verify-email"; got != want {
		t.Fatalf("expected path %q, got %q", want, got)
	}
	if verifyURL.RawQuery != "" {
		t.Fatalf("expected no query string, got %q", verifyURL.RawQuery)
	}
	fragment, err := url.ParseQuery(verifyURL.Fragment)
	if err != nil {
		t.Fatalf("parse verification fragment: %v", err)
	}
	claims, err := tokens.Parse(fragment.Get("token"))
	if err != nil {
		t.Fatalf("parse verification token: %v", err)
	}
	if got, want := claims.Subject, user.ID; got != want {
		t.Fatalf("expected token subject %q, got %q", want, got)
	}
}

func TestSendIncludesOTP(t *testing.T) {
	service, store, mailer, _ := newService(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	store.users[user.ID] = user

	if err := service.Send(context.Background(), user); err != nil {
		t.Fatalf("send verification: %v", err)
	}
	if len(mailer.code) != 6 {
		t.Fatalf("expected a 6-digit code, got %q", mailer.code)
	}
}

func TestVerifyOTP(t *testing.T) {
	service, store, mailer, _ := newService(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	store.users[user.ID] = user

	if err := service.Send(context.Background(), user); err != nil {
		t.Fatalf("send verification: %v", err)
	}

	verified, err := service.VerifyOTP(context.Background(), user.Email, mailer.code)
	if err != nil {
		t.Fatalf("verify otp: %v", err)
	}
	if !verified.EmailVerified {
		t.Fatal("expected the email to be verified")
	}
	if store.otpHash != "" {
		t.Fatal("expected the otp to be cleared")
	}
}

func TestVerifyOTPRejectsWrongCode(t *testing.T) {
	service, store, mailer, _ := newService(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	store.users[user.ID] = user

	if err := service.Send(context.Background(), user); err != nil {
		t.Fatalf("send verification: %v", err)
	}

	wrong := "000000"
	if mailer.code == wrong {
		wrong = "000001"
	}
	if _, err := service.VerifyOTP(context.Background(), user.Email, wrong); !errors.Is(err, entities.ErrInvalidVerificationCode) {
		t.Fatalf("expected ErrInvalidVerificationCode, got %v", err)
	}
	if store.otpAttempts != 1 {
		t.Fatalf("expected one attempt, got %d", store.otpAttempts)
	}
}

func TestVerifyOTPRejectsExpiredCode(t *testing.T) {
	service, store, mailer, _ := newService(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	store.users[user.ID] = user

	if err := service.Send(context.Background(), user); err != nil {
		t.Fatalf("send verification: %v", err)
	}
	store.otpExpires = time.Now().UTC().Add(-time.Minute)

	if _, err := service.VerifyOTP(context.Background(), user.Email, mailer.code); !errors.Is(err, entities.ErrInvalidVerificationCode) {
		t.Fatalf("expected ErrInvalidVerificationCode, got %v", err)
	}
}

func TestVerifyOTPRejectsTooManyAttempts(t *testing.T) {
	service, store, mailer, _ := newService(t)
	user := entities.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
	store.users[user.ID] = user

	if err := service.Send(context.Background(), user); err != nil {
		t.Fatalf("send verification: %v", err)
	}
	store.otpAttempts = 5

	if _, err := service.VerifyOTP(context.Background(), user.Email, mailer.code); !errors.Is(err, entities.ErrTooManyVerificationAttempts) {
		t.Fatalf("expected ErrTooManyVerificationAttempts, got %v", err)
	}
}

func TestVerifyOTPAlreadyVerified(t *testing.T) {
	service, store, _, _ := newService(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com", EmailVerified: true}
	store.users[user.ID] = user

	verified, err := service.VerifyOTP(context.Background(), user.Email, "123456")
	if err != nil {
		t.Fatalf("verify otp: %v", err)
	}
	if !verified.EmailVerified {
		t.Fatal("expected the verified user")
	}
}

func TestVerifyOTPUnknownEmail(t *testing.T) {
	service, _, _, _ := newService(t)

	if _, err := service.VerifyOTP(context.Background(), "missing@example.com", "123456"); !errors.Is(err, entities.ErrInvalidVerificationCode) {
		t.Fatalf("expected ErrInvalidVerificationCode, got %v", err)
	}
}

func TestSendMailerFailure(t *testing.T) {
	service, store, mailer, _ := newService(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com"}
	store.users[user.ID] = user
	mailer.err = errors.New("smtp down")

	err := service.Send(context.Background(), user)
	if !errors.Is(err, services.ErrEmailDelivery) {
		t.Fatalf("expected ErrEmailDelivery, got %v", err)
	}
}

func TestVerify(t *testing.T) {
	service, store, _, tokens := newService(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com"}
	store.users[user.ID] = user

	raw := issue(t, tokens, user)
	verified, err := service.Verify(context.Background(), raw)
	if err != nil {
		t.Fatalf("verify email: %v", err)
	}
	if verified.ID != user.ID || !verified.EmailVerified {
		t.Fatalf("expected verified user %+v, got %+v", user, verified)
	}
	if got := store.users[user.ID]; !got.EmailVerified || got.EmailVerifiedAt == nil {
		t.Fatalf("expected verified user, got %+v", got)
	}
}

func TestVerifyAlreadyVerified(t *testing.T) {
	service, store, _, tokens := newService(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com", EmailVerified: true}
	store.users[user.ID] = user

	raw := issue(t, tokens, user)
	verified, err := service.Verify(context.Background(), raw)
	if err != nil {
		t.Fatalf("verify email: %v", err)
	}
	if verified.ID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, verified.ID)
	}
	if len(store.marked) != 0 {
		t.Fatalf("expected no mark call, got %v", store.marked)
	}
}

func TestVerifyRejectsMismatchedEmail(t *testing.T) {
	service, store, _, tokens := newService(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com"}
	store.users[user.ID] = user
	raw := issue(t, tokens, user)

	changed := user
	changed.Email = "grace@example.com"
	store.users[user.ID] = changed

	if _, err := service.Verify(context.Background(), raw); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerifyRejectsUnknownUser(t *testing.T) {
	service, _, _, tokens := newService(t)
	user := entities.User{ID: "user-1", Email: "ada@example.com"}
	raw := issue(t, tokens, user)

	if _, err := service.Verify(context.Background(), raw); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerifyRejectsMalformedToken(t *testing.T) {
	service, _, _, _ := newService(t)

	if _, err := service.Verify(context.Background(), "not-a-token"); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestResend(t *testing.T) {
	tests := []struct {
		name       string
		user       *entities.User
		wantMail   bool
		wantSentAt bool
	}{
		{
			name: "unknown email",
		},
		{
			name: "already verified",
			user: &entities.User{ID: "user-1", Email: "ada@example.com", EmailVerified: true},
		},
		{
			name:       "throttled",
			user:       &entities.User{ID: "user-1", Email: "ada@example.com", VerificationSentAt: timePtr(time.Now().UTC())},
			wantSentAt: true,
		},
		{
			name:       "eligible",
			user:       &entities.User{ID: "user-1", Email: "ada@example.com"},
			wantMail:   true,
			wantSentAt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, store, mailer, _ := newService(t)
			if tt.user != nil {
				store.users[tt.user.ID] = *tt.user
			}

			if err := service.Resend(context.Background(), "ada@example.com"); err != nil {
				t.Fatalf("resend verification: %v", err)
			}
			if got := mailer.calls == 1; got != tt.wantMail {
				t.Fatalf("expected mail sent %t, got %t", tt.wantMail, got)
			}
			if tt.user != nil {
				if got := store.users[tt.user.ID].VerificationSentAt != nil; got != tt.wantSentAt {
					t.Fatalf("expected sent timestamp %t, got %t", tt.wantSentAt, got)
				}
			}
		})
	}
}

func TestResendRejectsInvalidEmail(t *testing.T) {
	service, _, _, _ := newService(t)

	err := service.Resend(context.Background(), "not-an-email")
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestResendMailFailure(t *testing.T) {
	service, store, mailer, _ := newService(t)
	store.users["user-1"] = entities.User{ID: "user-1", Email: "ada@example.com"}
	mailer.err = errors.New("smtp down")

	if err := service.Resend(context.Background(), "ada@example.com"); !errors.Is(err, services.ErrEmailDelivery) {
		t.Fatalf("expected ErrEmailDelivery, got %v", err)
	}
}

type fakeStore struct {
	users       map[string]entities.User
	marked      []string
	otpHash     string
	otpExpires  time.Time
	otpAttempts int
}

func (s *fakeStore) FindByID(_ context.Context, id string) (entities.User, error) {
	user, ok := s.users[id]
	if !ok {
		return entities.User{}, entities.ErrUserNotFound
	}
	return user, nil
}

func (s *fakeStore) FindByEmail(_ context.Context, email string) (entities.User, error) {
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return entities.User{}, entities.ErrUserNotFound
}

func (s *fakeStore) MarkEmailVerified(_ context.Context, id, email string, verifiedAt time.Time) (entities.User, error) {
	user, ok := s.users[id]
	if !ok || user.Email != email {
		return entities.User{}, entities.ErrUserNotFound
	}
	s.marked = append(s.marked, id)
	user.EmailVerified = true
	user.EmailVerifiedAt = &verifiedAt
	s.users[id] = user
	s.otpHash = ""
	s.otpExpires = time.Time{}
	s.otpAttempts = 0
	return user, nil
}

func (s *fakeStore) SetVerificationSentAt(_ context.Context, id string, sentAt time.Time) error {
	user, ok := s.users[id]
	if !ok {
		return entities.ErrUserNotFound
	}
	user.VerificationSentAt = &sentAt
	s.users[id] = user
	return nil
}

func (s *fakeStore) SetVerificationOTP(_ context.Context, id, hash string, expiresAt time.Time) error {
	if _, ok := s.users[id]; !ok {
		return entities.ErrUserNotFound
	}
	s.otpHash = hash
	s.otpExpires = expiresAt
	s.otpAttempts = 0
	return nil
}

func (s *fakeStore) VerificationOTP(_ context.Context, email string) (dao.VerificationOTP, error) {
	for id, user := range s.users {
		if user.Email == email {
			return dao.VerificationOTP{
				UserID:    id,
				Hash:      s.otpHash,
				ExpiresAt: s.otpExpires,
				Attempts:  s.otpAttempts,
			}, nil
		}
	}
	return dao.VerificationOTP{}, entities.ErrUserNotFound
}

func (s *fakeStore) IncrementVerificationOTPAttempts(_ context.Context, id string) (int, error) {
	if _, ok := s.users[id]; !ok {
		return 0, entities.ErrUserNotFound
	}
	s.otpAttempts++
	return s.otpAttempts, nil
}

func (s *fakeStore) ClearVerificationOTP(_ context.Context, id string) error {
	if _, ok := s.users[id]; !ok {
		return entities.ErrUserNotFound
	}
	s.otpHash = ""
	s.otpExpires = time.Time{}
	s.otpAttempts = 0
	return nil
}

type fakeMailer struct {
	calls     int
	to        string
	name      string
	verifyURL string
	code      string
	err       error
}

func (m *fakeMailer) SendVerification(_ context.Context, to, name, verifyURL, code string) error {
	m.calls++
	m.to = to
	m.name = name
	m.verifyURL = verifyURL
	m.code = code
	return m.err
}

func newService(t *testing.T) (*services.EmailVerificationService, *fakeStore, *fakeMailer, *services.VerificationTokens) {
	t.Helper()

	tokens := newTokens(t)
	store := &fakeStore{users: make(map[string]entities.User)}
	mailer := &fakeMailer{}
	service := services.NewEmailVerificationService(store, tokens, mailer, "https://api.example.com/")
	return service, store, mailer, tokens
}

func newTokens(t *testing.T) *services.VerificationTokens {
	t.Helper()
	return newTokensWithSecret(t, strings.Repeat("s", 32))
}

func newTokensWithSecret(t *testing.T, secret string) *services.VerificationTokens {
	t.Helper()

	tokens, err := services.NewVerificationTokens(secret)
	if err != nil {
		t.Fatalf("create tokens: %v", err)
	}
	return tokens
}

func issue(t *testing.T, tokens *services.VerificationTokens, user entities.User) string {
	t.Helper()

	raw, err := tokens.Issue(user, time.Now().UTC())
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return raw
}

func timePtr(t time.Time) *time.Time {
	return &t
}
