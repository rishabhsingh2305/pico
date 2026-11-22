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
			collection: credentialsCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "tokenHash", Value: 1}},
					Options: options.Index().SetName("credentials_token_hash_unique").SetUnique(true),
				},
				{
					Keys: bson.D{
						{Key: "orgId", Value: 1},
						{Key: "kind", Value: 1},
						{Key: "resourceType", Value: 1},
						{Key: "resourceId", Value: 1},
						{Key: "revokedAt", Value: 1},
						{Key: "expiresAt", Value: 1},
					},
					Options: options.Index().SetName("credentials_lookup"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("credentials_org_created"),
				},
			},
		},
		{
			collection: linksCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "slug", Value: 1}},
					Options: options.Index().SetName("links_slug_unique").SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("links_org_created"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "tags", Value: 1}},
					Options: options.Index().SetName("links_org_tags"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "slug", Value: 1}},
					Options: options.Index().SetName("links_org_slug"),
				},
			},
		},
		{
			collection: linkClicksCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "linkId", Value: 1}, {Key: "hourStart", Value: 1}},
					Options: options.Index().SetName("link_clicks_link_hour_unique").SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "linkId", Value: 1}, {Key: "hourStart", Value: 1}},
					Options: options.Index().SetName("link_clicks_org_link_hour"),
				},
			},
		},
		{
			collection: linkAnalyticsStateCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "pullZoneId", Value: 1}},
					Options: options.Index().SetName("link_analytics_state_pull_zone_unique").SetUnique(true),
				},
			},
		},
		{
			collection: analyticsSourcesCollection,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "key", Value: 1}},
					Options: options.Index().SetName("analytics_sources_key_unique").SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}},
					Options: options.Index().SetName("analytics_sources_org_created"),
				},
			},
		},
		{
			collection: analyticsCapturesCollection,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "sourceId", Value: 1},
						{Key: "event", Value: 1},
						{Key: "hourStart", Value: 1},
						{Key: "countryCode", Value: 1},
						{Key: "device", Value: 1},
						{Key: "isBot", Value: 1},
						{Key: "referrerHost", Value: 1},
					},
					Options: options.Index().SetName("analytics_captures_dimension_unique").SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "sourceId", Value: 1}, {Key: "event", Value: 1}, {Key: "hourStart", Value: 1}},
					Options: options.Index().SetName("analytics_captures_org_source_event_hour"),
				},
				{
					Keys:    bson.D{{Key: "orgId", Value: 1}, {Key: "sourceId", Value: 1}, {Key: "hourStart", Value: 1}},
					Options: options.Index().SetName("analytics_captures_org_source_hour"),
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
