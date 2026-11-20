package mongodb

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

const linksCollection = "links"

// LinkStore persists short links in MongoDB.
type LinkStore struct {
	collection *mongo.Collection
}

var _ dao.LinkDAO = (*LinkStore)(nil)

// NewLinkStore returns a link store backed by client.
func NewLinkStore(client *Client) *LinkStore {
	return &LinkStore{collection: client.collection(linksCollection)}
}

type linkDocument struct {
	ID         bson.ObjectID `bson:"_id"`
	OrgID      string        `bson:"orgId"`
	UserID     string        `bson:"userId"`
	Slug       string        `bson:"slug"`
	URL        string        `bson:"url"`
	Title      string        `bson:"title"`
	Tags       []string      `bson:"tags"`
	DisabledAt *time.Time    `bson:"disabledAt,omitempty"`
	DeletedAt  *time.Time    `bson:"deletedAt,omitempty"`
	CreatedAt  time.Time     `bson:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt"`
}

func (d linkDocument) link() entities.Link {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}

	return entities.Link{
		ID:         d.ID.Hex(),
		OrgID:      d.OrgID,
		UserID:     d.UserID,
		Slug:       d.Slug,
		URL:        d.URL,
		Title:      d.Title,
		Tags:       tags,
		DisabledAt: d.DisabledAt,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

// Insert stores a link and returns it with its generated ID. A duplicate slug
// is reported as entities.ErrSlugTaken.
func (s *LinkStore) Insert(ctx context.Context, link entities.Link) (entities.Link, error) {
	tags := link.Tags
	if tags == nil {
		tags = []string{}
	}

	document := linkDocument{
		ID:         bson.NewObjectID(),
		OrgID:      link.OrgID,
		UserID:     link.UserID,
		Slug:       link.Slug,
		URL:        link.URL,
		Title:      link.Title,
		Tags:       tags,
		DisabledAt: link.DisabledAt,
		CreatedAt:  link.CreatedAt,
		UpdatedAt:  link.UpdatedAt,
	}

	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return entities.Link{}, entities.ErrSlugTaken
		}
		return entities.Link{}, fmt.Errorf("insert link: %w", err)
	}
	return document.link(), nil
}

// FindByID returns a link scoped to orgID. Disabled links are returned for
// management; deleted links are not.
func (s *LinkStore) FindByID(ctx context.Context, orgID, id string) (entities.Link, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Link{}, entities.ErrLinkNotFound
	}

	var document linkDocument
	err = s.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"orgId":     orgID,
		"deletedAt": nil,
	}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Link{}, entities.ErrLinkNotFound
	}
	if err != nil {
		return entities.Link{}, fmt.Errorf("find link: %w", err)
	}
	return document.link(), nil
}

// FindBySlug returns an active link for a public redirect.
func (s *LinkStore) FindBySlug(ctx context.Context, slug string) (entities.Link, error) {
	var document linkDocument
	err := s.collection.FindOne(ctx, bson.M{
		"slug":       slug,
		"disabledAt": nil,
		"deletedAt":  nil,
	}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Link{}, entities.ErrLinkNotFound
	}
	if err != nil {
		return entities.Link{}, fmt.Errorf("find link by slug: %w", err)
	}
	return document.link(), nil
}

// List returns links for an organization ordered by creation time. Tags use
// any-match semantics and Query is an alias prefix.
func (s *LinkStore) List(ctx context.Context, filter dao.LinkFilter) ([]entities.Link, error) {
	query := bson.M{
		"orgId":     filter.OrgID,
		"deletedAt": nil,
	}

	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}
	if filter.Query != "" {
		query["slug"] = bson.M{
			"$regex": "^" + regexp.QuoteMeta(strings.ToLower(filter.Query)),
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(filter.Limit)).
		SetSkip(int64(filter.Offset))

	cursor, err := s.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}

	var documents []linkDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode links: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close link cursor: %w", err)
	}

	items := make([]entities.Link, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.link())
	}
	return items, nil
}

// Update applies optional fields to a link scoped to orgID.
func (s *LinkStore) Update(ctx context.Context, orgID, id string, update dao.LinkUpdate) (entities.Link, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Link{}, entities.ErrLinkNotFound
	}

	set := bson.M{"updatedAt": update.UpdatedAt}
	unset := bson.M{}
	if update.URL != nil {
		set["url"] = *update.URL
	}
	if update.Title != nil {
		set["title"] = *update.Title
	}
	if update.Tags != nil {
		set["tags"] = *update.Tags
	}
	if update.Disabled != nil {
		if *update.Disabled {
			set["disabledAt"] = update.UpdatedAt
		} else {
			unset["disabledAt"] = ""
		}
	}

	updateDocument := bson.M{"$set": set}
	if len(unset) > 0 {
		updateDocument["$unset"] = unset
	}

	var document linkDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID, "deletedAt": nil},
		updateDocument,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Link{}, entities.ErrLinkNotFound
	}
	if err != nil {
		return entities.Link{}, fmt.Errorf("update link: %w", err)
	}
	return document.link(), nil
}

// Delete tombstones a link scoped to orgID. The slug stays reserved.
func (s *LinkStore) Delete(ctx context.Context, orgID, id string, deletedAt time.Time) (entities.Link, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Link{}, entities.ErrLinkNotFound
	}

	var document linkDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID, "deletedAt": nil},
		bson.M{"$set": bson.M{
			"deletedAt": deletedAt,
			"updatedAt": deletedAt,
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Link{}, entities.ErrLinkNotFound
	}
	if err != nil {
		return entities.Link{}, fmt.Errorf("delete link: %w", err)
	}
	return document.link(), nil
}
