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

const orgsCollection = "orgs"

// OrgStore persists organizations in MongoDB.
type OrgStore struct {
	collection *mongo.Collection
}

var _ dao.OrgDAO = (*OrgStore)(nil)

// NewOrgStore returns an organization store backed by client.
func NewOrgStore(client *Client) *OrgStore {
	return &OrgStore{collection: client.collection(orgsCollection)}
}

type orgDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	UserID    string        `bson:"userId"`
	Tags      []string      `bson:"tags"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

func (d orgDocument) org() entities.Org {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	return entities.Org{
		ID:        d.ID.Hex(),
		Name:      d.Name,
		UserID:    d.UserID,
		Tags:      tags,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// Insert stores an organization and returns it with its generated ID.
func (s *OrgStore) Insert(ctx context.Context, org entities.Org) (entities.Org, error) {
	tags := org.Tags
	if tags == nil {
		tags = []string{}
	}

	document := orgDocument{
		ID:        bson.NewObjectID(),
		Name:      org.Name,
		UserID:    org.UserID,
		Tags:      tags,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}

	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		return entities.Org{}, fmt.Errorf("insert org: %w", err)
	}
	return document.org(), nil
}

// FindByID returns an organization by ID.
func (s *OrgStore) FindByID(ctx context.Context, id string) (entities.Org, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Org{}, entities.ErrOrgNotFound
	}

	var document orgDocument
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Org{}, entities.ErrOrgNotFound
	}
	if err != nil {
		return entities.Org{}, fmt.Errorf("find org: %w", err)
	}
	return document.org(), nil
}

// FindByIDs returns organizations matching ids and, when tags are supplied,
// at least one of the tags. Invalid IDs are ignored.
func (s *OrgStore) FindByIDs(ctx context.Context, ids []string, tags []string) ([]entities.Org, error) {
	objectIDs := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		objectID, err := bson.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}
	if len(objectIDs) == 0 {
		return []entities.Org{}, nil
	}

	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	if len(tags) > 0 {
		filter["tags"] = bson.M{"$in": tags}
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find orgs: %w", err)
	}

	var documents []orgDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode orgs: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close org cursor: %w", err)
	}

	items := make([]entities.Org, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.org())
	}
	return items, nil
}

// Update applies optional name and tag changes to an organization.
func (s *OrgStore) Update(ctx context.Context, id string, update dao.OrgUpdate) (entities.Org, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Org{}, entities.ErrOrgNotFound
	}

	set := bson.M{"updatedAt": update.UpdatedAt}
	if update.Name != nil {
		set["name"] = *update.Name
	}
	if update.Tags != nil {
		set["tags"] = *update.Tags
	}

	var document orgDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Org{}, entities.ErrOrgNotFound
	}
	if err != nil {
		return entities.Org{}, fmt.Errorf("update org: %w", err)
	}
	return document.org(), nil
}

// Delete removes an organization. It is used for best-effort cleanup when
// owner membership creation fails.
func (s *OrgStore) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil
	}

	if _, err := s.collection.DeleteOne(ctx, bson.M{"_id": objectID}); err != nil {
		return fmt.Errorf("delete org: %w", err)
	}
	return nil
}

// Exists reports whether an organization exists.
func (s *OrgStore) Exists(ctx context.Context, id string) (bool, error) {
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
		return false, fmt.Errorf("check org: %w", err)
	}
	return true, nil
}

// OwnerID returns the owner of an organization. Found is false when the org
// does not exist.
func (s *OrgStore) OwnerID(ctx context.Context, id string) (string, bool, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return "", false, nil
	}

	var document struct {
		UserID string `bson:"userId"`
	}
	err = s.collection.FindOne(
		ctx,
		bson.M{"_id": objectID},
		options.FindOne().SetProjection(bson.M{"userId": 1}),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("find org owner: %w", err)
	}
	return document.UserID, true, nil
}
