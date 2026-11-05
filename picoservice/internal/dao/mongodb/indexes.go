package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes creates the indexes required for correctness and pagination.
// It is idempotent and is run by cmd/indexes rather than during Lambda startup.
func (c *Client) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{
			collection: usersCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "email", Value: 1}},
					Options: options.Index().SetName("users_email_unique").SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("users_created_at"),
				},
			},
		},
		{
			collection: orgsCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("orgs_user_created"),
				},
			},
		},
		{
			collection: membershipsCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "userId", Value: 1}},
					Options: options.Index().SetName("memberships_org_user_unique").SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("memberships_user_created"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("memberships_org_created"),
				},
			},
		},
		{
			collection: projectsCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("projects_org_created"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "tags", Value: 1}},
					Options: options.Index().SetName("projects_org_tags"),
				},
			},
		},
		{
			collection: expensesCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "spentAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("expenses_org_spent"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "projectId", Value: 1}, {Key: "spentAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("expenses_org_project_spent"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "tags", Value: 1}},
					Options: options.Index().SetName("expenses_org_tags"),
				},
			},
		},
	}

	for _, index := range indexes {
		if _, err := c.collection(index.collection).Indexes().CreateMany(ctx, index.models); err != nil {
			return fmt.Errorf("create %s indexes: %w", index.collection, err)
		}
	}
	return nil
}
