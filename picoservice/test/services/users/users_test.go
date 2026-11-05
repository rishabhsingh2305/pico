package userstest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

func TestCreateNormalizesAndHashes(t *testing.T) {
	store := newFakeStore()
	hasher := &fakeHasher{}
	verifier := &fakeVerification{}
	service := services.NewUserService(store, hasher, verifier)

	user, err := service.Create(context.Background(), services.CreateUserInput{
		Name:     "  Ada Lovelace  ",
		Email:    "  ADA@EXAMPLE.COM ",
		Password: "correct horse",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if got, want := user.Name, "Ada Lovelace"; got != want {
		t.Fatalf("expected name %q, got %q", want, got)
	}
	if got, want := user.Email, "ada@example.com"; got != want {
		t.Fatalf("expected email %q, got %q", want, got)
	}
	if got, want := store.hashes[user.ID], "hashed:correct horse"; got != want {
		t.Fatalf("expected stored hash %q, got %q", want, got)
	}
	if hasher.lastPassword != "correct horse" {
		t.Fatalf("expected hasher to receive the original password, got %q", hasher.lastPassword)
	}
	if len(verifier.sent) != 1 || verifier.sent[0].ID != user.ID {
		t.Fatalf("expected verification email for %q, got %+v", user.ID, verifier.sent)
	}
}

func TestCreateRollsBackWhenVerificationFails(t *testing.T) {
	store := newFakeStore()
	verifier := &fakeVerification{sendErr: services.ErrEmailDelivery}
	service := services.NewUserService(store, &fakeHasher{}, verifier)

	_, err := service.Create(context.Background(), services.CreateUserInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "correct horse",
	})
	if !errors.Is(err, services.ErrEmailDelivery) {
		t.Fatalf("expected ErrEmailDelivery, got %v", err)
	}
	if len(store.users) != 0 {
		t.Fatalf("expected the user to be rolled back, got %+v", store.users)
	}
}

func TestUpdateEmailResetsVerification(t *testing.T) {
	store := newFakeStore()
	user := store.addUser("user-1", "Ada", "ada@example.com")
	user.EmailVerified = true
	store.users[user.ID] = user

	verifier := &fakeVerification{}
	service := services.NewUserService(store, &fakeHasher{}, verifier)

	email := "grace@example.com"
	updated, err := service.Update(context.Background(), services.UpdateUserInput{
		ActorID: user.ID,
		UserID:  user.ID,
		Email:   &email,
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if updated.EmailVerified {
		t.Fatal("expected email verification to be reset")
	}
	if len(verifier.sent) != 1 || verifier.sent[0].Email != email {
		t.Fatalf("expected a verification email for %q, got %+v", email, verifier.sent)
	}
}

func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name  string
		input services.CreateUserInput
		field string
	}{
		{
			name:  "missing name",
			input: services.CreateUserInput{Email: "ada@example.com", Password: "correct horse"},
			field: "name",
		},
		{
			name:  "invalid email",
			input: services.CreateUserInput{Name: "Ada", Email: "not-an-email", Password: "correct horse"},
			field: "email",
		},
		{
			name:  "short password",
			input: services.CreateUserInput{Name: "Ada", Email: "ada@example.com", Password: "short"},
			field: "password",
		},
		{
			name:  "long password",
			input: services.CreateUserInput{Name: "Ada", Email: "ada@example.com", Password: string(make([]byte, 73))},
			field: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewUserService(newFakeStore(), &fakeHasher{}, &fakeVerification{})

			_, err := service.Create(context.Background(), tt.input)
			var validationErr *validation.Error
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("expected field %q, got %q", tt.field, validationErr.Field)
			}
		})
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	store := newFakeStore()
	store.insertErr = entities.ErrEmailTaken
	service := services.NewUserService(store, &fakeHasher{}, &fakeVerification{})

	_, err := service.Create(context.Background(), services.CreateUserInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "correct horse",
	})
	if !errors.Is(err, entities.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestUpdateOnlySelf(t *testing.T) {
	store := newFakeStore()
	user := store.addUser("user-1", "Ada", "ada@example.com")
	service := services.NewUserService(store, &fakeHasher{}, &fakeVerification{})

	name := "Grace"
	_, err := service.Update(context.Background(), services.UpdateUserInput{
		ActorID: "user-2",
		UserID:  user.ID,
		Name:    &name,
	})
	if !errors.Is(err, entities.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if got := store.users[user.ID].Name; got != "Ada" {
		t.Fatalf("expected unchanged name, got %q", got)
	}
}

func TestUpdateRequiresFields(t *testing.T) {
	store := newFakeStore()
	user := store.addUser("user-1", "Ada", "ada@example.com")
	service := services.NewUserService(store, &fakeHasher{}, &fakeVerification{})

	_, err := service.Update(context.Background(), services.UpdateUserInput{
		ActorID: user.ID,
		UserID:  user.ID,
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateChangesContactFields(t *testing.T) {
	store := newFakeStore()
	user := store.addUser("user-1", "Ada", "ada@example.com")
	service := services.NewUserService(store, &fakeHasher{}, &fakeVerification{})

	name := "  Grace Hopper "
	email := " GRACE@EXAMPLE.COM "
	updated, err := service.Update(context.Background(), services.UpdateUserInput{
		ActorID: user.ID,
		UserID:  user.ID,
		Name:    &name,
		Email:   &email,
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if got, want := updated.Name, "Grace Hopper"; got != want {
		t.Fatalf("expected name %q, got %q", want, got)
	}
	if got, want := updated.Email, "grace@example.com"; got != want {
		t.Fatalf("expected email %q, got %q", want, got)
	}
}

func TestUpdateEmailSendFailure(t *testing.T) {
	store := newFakeStore()
	user := store.addUser("user-1", "Ada", "ada@example.com")
	verifier := &fakeVerification{sendErr: services.ErrEmailDelivery}
	service := services.NewUserService(store, &fakeHasher{}, verifier)

	email := "grace@example.com"
	_, err := service.Update(context.Background(), services.UpdateUserInput{
		ActorID: user.ID,
		UserID:  user.ID,
		Email:   &email,
	})
	if !errors.Is(err, services.ErrEmailDelivery) {
		t.Fatalf("expected ErrEmailDelivery, got %v", err)
	}
	if got, want := store.users[user.ID].Email, email; got != want {
		t.Fatalf("expected updated email %q, got %q", want, got)
	}
	if store.users[user.ID].EmailVerified {
		t.Fatal("expected the new email to remain unverified")
	}
}

func TestChangePassword(t *testing.T) {
	store := newFakeStore()
	user := store.addUser("user-1", "Ada", "ada@example.com")
	store.hashes[user.ID] = "hashed:old password"
	service := services.NewUserService(store, &fakeHasher{}, &fakeVerification{})

	t.Run("wrong current password", func(t *testing.T) {
		err := service.ChangePassword(context.Background(), services.ChangePasswordInput{
			ActorID:         user.ID,
			UserID:          user.ID,
			CurrentPassword: "wrong",
			NewPassword:     "new password",
		})
		if !errors.Is(err, entities.ErrInvalidPassword) {
			t.Fatalf("expected ErrInvalidPassword, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		err := service.ChangePassword(context.Background(), services.ChangePasswordInput{
			ActorID:         user.ID,
			UserID:          user.ID,
			CurrentPassword: "old password",
			NewPassword:     "new password",
		})
		if err != nil {
			t.Fatalf("change password: %v", err)
		}
		if got, want := store.hashes[user.ID], "hashed:new password"; got != want {
			t.Fatalf("expected hash %q, got %q", want, got)
		}
	})

	t.Run("only self", func(t *testing.T) {
		err := service.ChangePassword(context.Background(), services.ChangePasswordInput{
			ActorID:         "user-2",
			UserID:          user.ID,
			CurrentPassword: "old password",
			NewPassword:     "new password",
		})
		if !errors.Is(err, entities.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}

func TestListUsesPagination(t *testing.T) {
	store := newFakeStore()
	service := services.NewUserService(store, &fakeHasher{}, &fakeVerification{})

	if _, err := service.List(context.Background(), services.ListUsersInput{}); err != nil {
		t.Fatalf("list users: %v", err)
	}
	if got, want := store.lastLimit, validation.DefaultLimit; got != want {
		t.Fatalf("expected default limit %d, got %d", want, got)
	}
}

func TestBcryptHasher(t *testing.T) {
	hasher := services.NewBcryptHasher()

	hash, err := hasher.Hash("correct horse")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "correct horse" {
		t.Fatal("expected a bcrypt hash, got the plaintext password")
	}
	if err := hasher.Compare(hash, "correct horse"); err != nil {
		t.Fatalf("compare password: %v", err)
	}
	if err := hasher.Compare(hash, "wrong"); !errors.Is(err, entities.ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}
}

type fakeStore struct {
	users             map[string]entities.User
	hashes            map[string]string
	nextID            int
	insertErr         error
	lastLimit         int
	lastOffset        int
	updateErr         error
	updatePasswordErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:  make(map[string]entities.User),
		hashes: make(map[string]string),
	}
}

func (s *fakeStore) addUser(id, name, email string) entities.User {
	now := time.Now().UTC()
	user := entities.User{ID: id, Name: name, Email: email, CreatedAt: now, UpdatedAt: now}
	s.users[id] = user
	return user
}

func (s *fakeStore) Insert(_ context.Context, user entities.User, passwordHash string) (entities.User, error) {
	if s.insertErr != nil {
		return entities.User{}, s.insertErr
	}
	s.nextID++
	user.ID = fmt.Sprintf("user-%d", s.nextID)
	s.users[user.ID] = user
	s.hashes[user.ID] = passwordHash
	return user, nil
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

func (s *fakeStore) PasswordHash(_ context.Context, id string) (string, error) {
	hash, ok := s.hashes[id]
	if !ok {
		return "", entities.ErrUserNotFound
	}
	return hash, nil
}

func (s *fakeStore) List(_ context.Context, limit, offset int) ([]entities.User, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	return []entities.User{}, nil
}

func (s *fakeStore) Update(_ context.Context, id, name, email string, emailVerified bool, updatedAt time.Time) (entities.User, error) {
	if s.updateErr != nil {
		return entities.User{}, s.updateErr
	}
	user, ok := s.users[id]
	if !ok {
		return entities.User{}, entities.ErrUserNotFound
	}
	user.Name = name
	user.Email = email
	user.EmailVerified = emailVerified
	if !emailVerified {
		user.EmailVerifiedAt = nil
	}
	user.UpdatedAt = updatedAt
	s.users[id] = user
	return user, nil
}

func (s *fakeStore) MarkEmailVerified(_ context.Context, id, email string, verifiedAt time.Time) (entities.User, error) {
	user, ok := s.users[id]
	if !ok || user.Email != email {
		return entities.User{}, entities.ErrUserNotFound
	}
	user.EmailVerified = true
	user.EmailVerifiedAt = &verifiedAt
	user.UpdatedAt = verifiedAt
	s.users[id] = user
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

func (s *fakeStore) Delete(_ context.Context, id string) error {
	if _, ok := s.users[id]; !ok {
		return entities.ErrUserNotFound
	}
	delete(s.users, id)
	delete(s.hashes, id)
	return nil
}

func (s *fakeStore) UpdatePassword(_ context.Context, id, passwordHash string, updatedAt time.Time) error {
	if s.updatePasswordErr != nil {
		return s.updatePasswordErr
	}
	if _, ok := s.users[id]; !ok {
		return entities.ErrUserNotFound
	}
	s.hashes[id] = passwordHash
	user := s.users[id]
	user.UpdatedAt = updatedAt
	s.users[id] = user
	return nil
}

func (s *fakeStore) Exists(_ context.Context, id string) (bool, error) {
	_, ok := s.users[id]
	return ok, nil
}

type fakeHasher struct {
	lastPassword string
}

func (h *fakeHasher) Hash(password string) (string, error) {
	h.lastPassword = password
	return "hashed:" + password, nil
}

func (h *fakeHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return entities.ErrPasswordMismatch
	}
	return nil
}

type fakeVerification struct {
	sent    []entities.User
	sendErr error
}

func (v *fakeVerification) Send(_ context.Context, user entities.User) error {
	if v.sendErr != nil {
		return v.sendErr
	}
	v.sent = append(v.sent, user)
	return nil
}

func (v *fakeVerification) Verify(context.Context, string) (entities.User, error) {
	return entities.User{}, nil
}

func (v *fakeVerification) Resend(context.Context, string) error {
	return nil
}
