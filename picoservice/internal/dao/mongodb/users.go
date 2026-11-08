package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

const usersCollection = "users"

// UserStore persists users in MongoDB.
type UserStore struct {
	collection *mongo.Collection
}

var _ dao.UserDAO = (*UserStore)(nil)

// NewUserStore returns a user store backed by client.
func NewUserStore(client *Client) *UserStore {
	return &UserStore{collection: client.collection(usersCollection)}
}

type userDocument struct {
	ID                      bson.ObjectID `bson:"_id"`
	Name                    string        `bson:"name"`
	Email                   string        `bson:"email"`
	PasswordHash            string        `bson:"passwordHash"`
	EmailVerified           bool          `bson:"emailVerified"`
	EmailVerifiedAt         *time.Time    `bson:"emailVerifiedAt,omitempty"`
	VerificationSentAt      *time.Time    `bson:"verificationSentAt,omitempty"`
	VerificationOTPHash     string        `bson:"verificationOTPHash,omitempty"`
	VerificationOTPExpires  *time.Time    `bson:"verificationOTPExpiresAt,omitempty"`
	VerificationOTPAttempts int           `bson:"verificationOTPAttempts,omitempty"`
	CreatedAt               time.Time     `bson:"createdAt"`
	UpdatedAt               time.Time     `bson:"updatedAt"`
}

func (d userDocument) user() entities.User {
	return entities.User{
		ID:                 d.ID.Hex(),
		Name:               d.Name,
		Email:              d.Email,
		EmailVerified:      d.EmailVerified,
		EmailVerifiedAt:    d.EmailVerifiedAt,
		VerificationSentAt: d.VerificationSentAt,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// Insert stores a user and its password hash.
func (s *UserStore) Insert(ctx context.Context, user entities.User, passwordHash string) (entities.User, error) {
	document := userDocument{
		ID:                 bson.NewObjectID(),
		Name:               user.Name,
		Email:              user.Email,
		PasswordHash:       passwordHash,
		EmailVerified:      user.EmailVerified,
		EmailVerifiedAt:    user.EmailVerifiedAt,
		VerificationSentAt: user.VerificationSentAt,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}

	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return entities.User{}, entities.ErrEmailTaken
		}
		return entities.User{}, fmt.Errorf("insert user: %w", err)
	}
	return document.user(), nil
}

// FindByID returns a user by ID.
func (s *UserStore) FindByID(ctx context.Context, id string) (entities.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.User{}, entities.ErrUserNotFound
	}

	var document userDocument
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.User{}, entities.ErrUserNotFound
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("find user: %w", err)
	}
	return document.user(), nil
}

// FindByEmail returns a user by normalized email address.
func (s *UserStore) FindByEmail(ctx context.Context, email string) (entities.User, error) {
	var document userDocument
	err := s.collection.FindOne(ctx, bson.M{"email": email}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.User{}, entities.ErrUserNotFound
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return document.user(), nil
}

// PasswordHash returns the stored password hash for a user.
func (s *UserStore) PasswordHash(ctx context.Context, id string) (string, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return "", entities.ErrUserNotFound
	}

	var document struct {
		PasswordHash string `bson:"passwordHash"`
	}
	err = s.collection.FindOne(
		ctx,
		bson.M{"_id": objectID},
		options.FindOne().SetProjection(bson.M{"passwordHash": 1}),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", entities.ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find password hash: %w", err)
	}
	return document.PasswordHash, nil
}

// List returns users ordered by creation time.
func (s *UserStore) List(ctx context.Context, limit, offset int) ([]entities.User, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := s.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	var documents []userDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode users: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close user cursor: %w", err)
	}

	items := make([]entities.User, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.user())
	}
	return items, nil
}

// Update changes a user's name and email. Changing the email clears email
// verification.
func (s *UserStore) Update(ctx context.Context, id, name, email string, emailVerified bool, updatedAt time.Time) (entities.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.User{}, entities.ErrUserNotFound
	}

	fields := bson.M{
		"name":          name,
		"email":         email,
		"emailVerified": emailVerified,
		"updatedAt":     updatedAt,
	}
	if !emailVerified {
		fields["emailVerifiedAt"] = nil
	}

	var document userDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": fields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.User{}, entities.ErrUserNotFound
	}
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return entities.User{}, entities.ErrEmailTaken
		}
		return entities.User{}, fmt.Errorf("update user: %w", err)
	}
	return document.user(), nil
}

// UpdatePassword replaces a user's password hash.
func (s *UserStore) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrUserNotFound
	}

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"passwordHash": passwordHash, "updatedAt": updatedAt}},
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if result.MatchedCount == 0 {
		return entities.ErrUserNotFound
	}
	return nil
}

// MarkEmailVerified records that a user's email address was verified.
func (s *UserStore) MarkEmailVerified(ctx context.Context, id, email string, verifiedAt time.Time) (entities.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.User{}, entities.ErrUserNotFound
	}

	var document userDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID, "email": email},
		bson.M{
			"$set": bson.M{
				"emailVerified":   true,
				"emailVerifiedAt": verifiedAt,
				"updatedAt":       verifiedAt,
			},
			"$unset": bson.M{
				"verificationOTPHash":      "",
				"verificationOTPExpiresAt": "",
				"verificationOTPAttempts":  "",
			},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.User{}, entities.ErrUserNotFound
	}
	if err != nil {
		return entities.User{}, fmt.Errorf("mark email verified: %w", err)
	}
	return document.user(), nil
}

// SetVerificationSentAt records when a verification email was last sent.
func (s *UserStore) SetVerificationSentAt(ctx context.Context, id string, sentAt time.Time) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrUserNotFound
	}

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"verificationSentAt": sentAt}},
	)
	if err != nil {
		return fmt.Errorf("set verification sent time: %w", err)
	}
	if result.MatchedCount == 0 {
		return entities.ErrUserNotFound
	}
	return nil
}

// SetVerificationOTP stores a hashed email verification code and its expiry.
func (s *UserStore) SetVerificationOTP(ctx context.Context, id, hash string, expiresAt time.Time) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrUserNotFound
	}

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{
			"verificationOTPHash":      hash,
			"verificationOTPExpiresAt": expiresAt,
			"verificationOTPAttempts":  0,
		}},
	)
	if err != nil {
		return fmt.Errorf("set verification otp: %w", err)
	}
	if result.MatchedCount == 0 {
		return entities.ErrUserNotFound
	}
	return nil
}

// VerificationOTP returns the stored verification code state for an email.
func (s *UserStore) VerificationOTP(ctx context.Context, email string) (dao.VerificationOTP, error) {
	var document struct {
		ID        bson.ObjectID `bson:"_id"`
		Hash      string        `bson:"verificationOTPHash"`
		ExpiresAt *time.Time    `bson:"verificationOTPExpiresAt"`
		Attempts  int           `bson:"verificationOTPAttempts"`
	}

	err := s.collection.FindOne(
		ctx,
		bson.M{"email": email},
		options.FindOne().SetProjection(bson.M{
			"_id":                      1,
			"verificationOTPHash":      1,
			"verificationOTPExpiresAt": 1,
			"verificationOTPAttempts":  1,
		}),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return dao.VerificationOTP{}, entities.ErrUserNotFound
	}
	if err != nil {
		return dao.VerificationOTP{}, fmt.Errorf("find verification otp: %w", err)
	}

	otp := dao.VerificationOTP{
		UserID:   document.ID.Hex(),
		Hash:     document.Hash,
		Attempts: document.Attempts,
	}
	if document.ExpiresAt != nil {
		otp.ExpiresAt = *document.ExpiresAt
	}
	return otp, nil
}

// IncrementVerificationOTPAttempts increments and returns the failed attempt
// count for a user.
func (s *UserStore) IncrementVerificationOTPAttempts(ctx context.Context, id string) (int, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return 0, entities.ErrUserNotFound
	}

	var document struct {
		Attempts int `bson:"verificationOTPAttempts"`
	}
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$inc": bson.M{"verificationOTPAttempts": 1}},
		options.FindOneAndUpdate().
			SetReturnDocument(options.After).
			SetProjection(bson.M{"verificationOTPAttempts": 1}),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, entities.ErrUserNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("increment verification otp attempts: %w", err)
	}
	return document.Attempts, nil
}

// ClearVerificationOTP removes a stored verification code.
func (s *UserStore) ClearVerificationOTP(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrUserNotFound
	}

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$unset": bson.M{
			"verificationOTPHash":      "",
			"verificationOTPExpiresAt": "",
			"verificationOTPAttempts":  "",
		}},
	)
	if err != nil {
		return fmt.Errorf("clear verification otp: %w", err)
	}
	if result.MatchedCount == 0 {
		return entities.ErrUserNotFound
	}
	return nil
}

// Delete removes a user. It exists for signup cleanup after email delivery
// fails and is not exposed through HTTP.
func (s *UserStore) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrUserNotFound
	}

	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.DeletedCount == 0 {
		return entities.ErrUserNotFound
	}
	return nil
}

// Exists reports whether a user exists.
func (s *UserStore) Exists(ctx context.Context, id string) (bool, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, nil
	}

	err = s.collection.FindOne(
		ctx,
		bson.M{"_id": objectID},
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check user: %w", err)
	}
	return true, nil
}
