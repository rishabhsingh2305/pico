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

const credentialsCollection = "credentials"

// CredentialStore persists share links and API keys in MongoDB.
type CredentialStore struct {
	collection *mongo.Collection
}

var _ dao.CredentialDAO = (*CredentialStore)(nil)

// NewCredentialStore returns a credential store backed by client.
func NewCredentialStore(client *Client) *CredentialStore {
	return &CredentialStore{collection: client.collection(credentialsCollection)}
}

type credentialDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Kind         string        `bson:"kind"`
	OrgID        string        `bson:"orgId"`
	ResourceType string        `bson:"resourceType,omitempty"`
	ResourceID   string        `bson:"resourceId,omitempty"`
	ProjectID    string        `bson:"projectId,omitempty"`
	Scopes       []string      `bson:"scopes"`
	Name         string        `bson:"name,omitempty"`
	TokenHash    string        `bson:"tokenHash"`
	CreatedBy    string        `bson:"createdBy"`
	CreatedAt    time.Time     `bson:"createdAt"`
	ExpiresAt    *time.Time    `bson:"expiresAt,omitempty"`
	RevokedAt    *time.Time    `bson:"revokedAt,omitempty"`
	LastUsedAt   *time.Time    `bson:"lastUsedAt,omitempty"`
}

func (d credentialDocument) credential() entities.Credential {
	return entities.Credential{
		ID:           d.ID.Hex(),
		Kind:         d.Kind,
		OrgID:        d.OrgID,
		ResourceType: d.ResourceType,
		ResourceID:   d.ResourceID,
		ProjectID:    d.ProjectID,
		Scopes:       d.Scopes,
		Name:         d.Name,
		TokenHash:    d.TokenHash,
		CreatedBy:    d.CreatedBy,
		CreatedAt:    d.CreatedAt,
		ExpiresAt:    d.ExpiresAt,
		RevokedAt:    d.RevokedAt,
		LastUsedAt:   d.LastUsedAt,
	}
}

// Insert stores a credential.
func (s *CredentialStore) Insert(ctx context.Context, credential entities.Credential) (entities.Credential, error) {
	document := credentialDocument{
		ID:           bson.NewObjectID(),
		Kind:         credential.Kind,
		OrgID:        credential.OrgID,
		ResourceType: credential.ResourceType,
		ResourceID:   credential.ResourceID,
		ProjectID:    credential.ProjectID,
		Scopes:       credential.Scopes,
		Name:         credential.Name,
		TokenHash:    credential.TokenHash,
		CreatedBy:    credential.CreatedBy,
		CreatedAt:    credential.CreatedAt,
		ExpiresAt:    credential.ExpiresAt,
		RevokedAt:    credential.RevokedAt,
		LastUsedAt:   credential.LastUsedAt,
	}
	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		return entities.Credential{}, fmt.Errorf("insert credential: %w", err)
	}
	return document.credential(), nil
}

// FindByID returns a credential scoped to an organization.
func (s *CredentialStore) FindByID(ctx context.Context, orgID, id string) (entities.Credential, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Credential{}, entities.ErrCredentialNotFound
	}

	var document credentialDocument
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID, "orgId": orgID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Credential{}, entities.ErrCredentialNotFound
	}
	if err != nil {
		return entities.Credential{}, fmt.Errorf("find credential: %w", err)
	}
	return document.credential(), nil
}

// FindByTokenHash returns a credential by its token hash.
func (s *CredentialStore) FindByTokenHash(ctx context.Context, tokenHash string) (entities.Credential, error) {
	var document credentialDocument
	err := s.collection.FindOne(ctx, bson.M{"tokenHash": tokenHash}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Credential{}, entities.ErrCredentialNotFound
	}
	if err != nil {
		return entities.Credential{}, fmt.Errorf("find credential by token: %w", err)
	}
	return document.credential(), nil
}

// List returns credentials matching filter.
func (s *CredentialStore) List(ctx context.Context, filter dao.CredentialFilter) ([]entities.Credential, error) {
	query := bson.M{"orgId": filter.OrgID}
	if filter.Kind != "" {
		query["kind"] = filter.Kind
	}
	if filter.ResourceType != "" {
		query["resourceType"] = filter.ResourceType
	}
	if len(filter.ResourceIDs) > 0 {
		query["resourceId"] = bson.M{"$in": filter.ResourceIDs}
	}
	if filter.Active {
		now := time.Now().UTC()
		query["revokedAt"] = nil
		query["$or"] = []bson.M{
			{"expiresAt": nil},
			{"expiresAt": bson.M{"$gt": now}},
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(filter.Limit)).
		SetSkip(int64(filter.Offset))

	cursor, err := s.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}

	var documents []credentialDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode credentials: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close credentials cursor: %w", err)
	}

	items := make([]entities.Credential, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.credential())
	}
	return items, nil
}

// Revoke marks a credential revoked.
func (s *CredentialStore) Revoke(ctx context.Context, orgID, id string, revokedAt time.Time) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrCredentialNotFound
	}

	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID},
		bson.M{"$set": bson.M{"revokedAt": revokedAt}},
	)
	if err != nil {
		return fmt.Errorf("revoke credential: %w", err)
	}
	if result.MatchedCount == 0 {
		return entities.ErrCredentialNotFound
	}
	return nil
}
