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

const membershipsCollection = "memberships"

// MembershipStore persists user-organization memberships in MongoDB.
type MembershipStore struct {
	collection *mongo.Collection
}

var _ dao.MembershipDAO = (*MembershipStore)(nil)

// NewMembershipStore returns a membership store backed by client.
func NewMembershipStore(client *Client) *MembershipStore {
	return &MembershipStore{collection: client.collection(membershipsCollection)}
}

type membershipDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	OrgID     string        `bson:"orgId"`
	UserID    string        `bson:"userId"`
	CreatedAt time.Time     `bson:"createdAt"`
}

func (d membershipDocument) membership() entities.Membership {
	return entities.Membership{
		ID:        d.ID.Hex(),
		OrgID:     d.OrgID,
		UserID:    d.UserID,
		CreatedAt: d.CreatedAt,
	}
}

// Insert stores a membership.
func (s *MembershipStore) Insert(ctx context.Context, membership entities.Membership) (entities.Membership, error) {
	document := membershipDocument{
		ID:        bson.NewObjectID(),
		OrgID:     membership.OrgID,
		UserID:    membership.UserID,
		CreatedAt: membership.CreatedAt,
	}

	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return entities.Membership{}, entities.ErrAlreadyMember
		}
		return entities.Membership{}, fmt.Errorf("insert membership: %w", err)
	}
	return document.membership(), nil
}

// AddOwner stores the owner's membership during org creation.
func (s *MembershipStore) AddOwner(ctx context.Context, orgID, userID string) error {
	_, err := s.Insert(ctx, entities.Membership{
		OrgID:     orgID,
		UserID:    userID,
		Role:      entities.RoleOwner,
		CreatedAt: time.Now().UTC(),
	})
	return err
}

// ListByOrg returns memberships for an organization ordered by creation time.
func (s *MembershipStore) ListByOrg(ctx context.Context, orgID string, limit, offset int) ([]entities.Membership, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := s.collection.Find(ctx, bson.M{"orgId": orgID}, opts)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}

	var documents []membershipDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode memberships: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close membership cursor: %w", err)
	}

	items := make([]entities.Membership, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.membership())
	}
	return items, nil
}

// ListOrgIDsByUser returns the IDs of organizations a user belongs to, ordered
// by membership creation time.
func (s *MembershipStore) ListOrgIDsByUser(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetProjection(bson.M{"orgId": 1})

	cursor, err := s.collection.Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("list user organizations: %w", err)
	}

	var documents []struct {
		OrgID string `bson:"orgId"`
	}
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode user organizations: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close user organization cursor: %w", err)
	}

	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.OrgID)
	}
	return ids, nil
}

// Exists reports whether a user belongs to an organization.
func (s *MembershipStore) Exists(ctx context.Context, orgID, userID string) (bool, error) {
	err := s.collection.FindOne(
		ctx,
		bson.M{"orgId": orgID, "userId": userID},
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return true, nil
}

// Delete removes a user from an organization.
func (s *MembershipStore) Delete(ctx context.Context, orgID, userID string) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"orgId": orgID, "userId": userID})
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	if result.DeletedCount == 0 {
		return entities.ErrMembershipNotFound
	}
	return nil
}
