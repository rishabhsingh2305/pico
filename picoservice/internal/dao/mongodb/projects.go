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

const projectsCollection = "projects"

// ProjectStore persists projects in MongoDB.
type ProjectStore struct {
	collection *mongo.Collection
}

var _ dao.ProjectDAO = (*ProjectStore)(nil)

// NewProjectStore returns a project store backed by client.
func NewProjectStore(client *Client) *ProjectStore {
	return &ProjectStore{collection: client.collection(projectsCollection)}
}

type projectDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	OrgID     string        `bson:"orgId"`
	UserID    string        `bson:"userId"`
	Name      string        `bson:"name"`
	Currency  string        `bson:"currency"`
	Tags      []string      `bson:"tags"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

func (d projectDocument) project() entities.Project {
	currency := d.Currency
	if currency == "" {
		currency = "USD"
	}
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	return entities.Project{
		ID:        d.ID.Hex(),
		OrgID:     d.OrgID,
		UserID:    d.UserID,
		Name:      d.Name,
		Currency:  currency,
		Tags:      tags,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// Insert stores a project and returns it with its generated ID.
func (s *ProjectStore) Insert(ctx context.Context, project entities.Project) (entities.Project, error) {
	tags := project.Tags
	if tags == nil {
		tags = []string{}
	}

	document := projectDocument{
		ID:        bson.NewObjectID(),
		OrgID:     project.OrgID,
		UserID:    project.UserID,
		Name:      project.Name,
		Currency:  project.Currency,
		Tags:      tags,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}

	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		return entities.Project{}, fmt.Errorf("insert project: %w", err)
	}
	return document.project(), nil
}

// FindByID returns a project scoped to orgID.
func (s *ProjectStore) FindByID(ctx context.Context, orgID, id string) (entities.Project, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Project{}, entities.ErrProjectNotFound
	}

	var document projectDocument
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID, "orgId": orgID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Project{}, entities.ErrProjectNotFound
	}
	if err != nil {
		return entities.Project{}, fmt.Errorf("find project: %w", err)
	}
	return document.project(), nil
}

// List returns projects for an organization ordered by creation time. Tags use
// any-match semantics.
func (s *ProjectStore) List(ctx context.Context, filter dao.ProjectFilter) ([]entities.Project, error) {
	query := bson.M{"orgId": filter.OrgID}
	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(filter.Limit)).
		SetSkip(int64(filter.Offset))

	cursor, err := s.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	var documents []projectDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode projects: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close project cursor: %w", err)
	}

	items := make([]entities.Project, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.project())
	}
	return items, nil
}

// Update applies optional fields to a project scoped to orgID.
func (s *ProjectStore) Update(ctx context.Context, orgID, id string, update dao.ProjectUpdate) (entities.Project, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Project{}, entities.ErrProjectNotFound
	}

	set := bson.M{"updatedAt": update.UpdatedAt}
	if update.Name != nil {
		set["name"] = *update.Name
	}
	if update.Tags != nil {
		set["tags"] = *update.Tags
	}

	var document projectDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Project{}, entities.ErrProjectNotFound
	}
	if err != nil {
		return entities.Project{}, fmt.Errorf("update project: %w", err)
	}
	return document.project(), nil
}

// Exists reports whether a project exists in orgID.
func (s *ProjectStore) Exists(ctx context.Context, orgID, id string) (bool, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, nil
	}

	err = s.collection.FindOne(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID},
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check project: %w", err)
	}
	return true, nil
}
