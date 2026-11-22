package mongodbtest

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"picoservice/internal/dao"
	"picoservice/internal/dao/mongodb"
	"picoservice/internal/entities"
)

func TestAnalyticsStoresIntegration(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to run MongoDB integration tests")
	}
	database := os.Getenv("MONGODB_TEST_DATABASE")
	if database == "" {
		database = "picoservice_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	raw, err := drivermongo.Connect(options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("connect raw client: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := clearAnalyticsCollections(cleanupCtx, raw, database); err != nil {
			t.Errorf("clear analytics collections: %v", err)
		}
		if err := raw.Disconnect(cleanupCtx); err != nil {
			t.Errorf("disconnect raw client: %v", err)
		}
	})
	if err := clearAnalyticsCollections(ctx, raw, database); err != nil {
		t.Fatalf("clear analytics collections: %v", err)
	}

	client, err := mongodb.New(mongodb.Config{URI: uri, Database: database})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Close(cleanupCtx); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	if err := client.EnsureIndexes(ctx); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}

	now := time.Now().UTC()
	store := mongodb.NewAnalyticsStore(client)
	source, err := store.Insert(ctx, entities.AnalyticsSource{
		OrgID:       "org-1",
		Name:        "Homepage",
		Description: "Hero pixel",
		Key:         "testpixelkey1234",
		Events:      []string{"view", "click"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	if _, err := store.Insert(ctx, entities.AnalyticsSource{
		OrgID: "org-1", Name: "Duplicate", Key: "testpixelkey1234",
		Events: []string{"view"}, CreatedAt: now, UpdatedAt: now,
	}); !errors.Is(err, entities.ErrAnalyticsKeyTaken) {
		t.Fatalf("expected ErrAnalyticsKeyTaken, got %v", err)
	}
	if _, err := store.Insert(ctx, entities.AnalyticsSource{
		OrgID: "org-1", Name: "Second", Key: "testpixelkey5678",
		Events: []string{"view"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert second source: %v", err)
	}
	found, err := store.FindByKeys(ctx, []string{"testpixelkey1234", "missing"})
	if err != nil || len(found) != 1 || found[0].ID != source.ID {
		t.Fatalf("find by keys: %+v, %v", found, err)
	}
	page, err := store.List(ctx, dao.AnalyticsSourceFilter{
		OrgID: "org-1", Query: "home", Limit: 20,
	})
	if err != nil || len(page) != 1 || page[0].ID != source.ID {
		t.Fatalf("list sources: %+v, %v", page, err)
	}

	hour := now.Truncate(time.Hour)
	if err := store.UpsertHourly(ctx, []dao.AnalyticsCaptureCount{
		{
			OrgID: "org-1", SourceID: source.ID, Event: "click", HourStart: hour,
			CountryCode: "IN", Device: "mobile", IsBot: false, ReferrerHost: "twitter.com", Count: 3,
		},
		{
			OrgID: "org-1", SourceID: source.ID, Event: "click", HourStart: hour,
			CountryCode: "IN", Device: "mobile", IsBot: false, ReferrerHost: "twitter.com", Count: 2,
		},
		{
			OrgID: "org-1", SourceID: source.ID, Event: "view", HourStart: hour.Add(time.Hour),
			CountryCode: "US", Device: "desktop", IsBot: true, ReferrerHost: "direct", Count: 4,
		},
	}); err != nil {
		t.Fatalf("upsert captures: %v", err)
	}

	stats, err := store.Stats(ctx, dao.AnalyticsCaptureFilter{
		OrgID: "org-1", SourceID: source.ID, Interval: "hour",
		From: hour, To: hour.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Total != 7 || len(stats.Series) != 2 || len(stats.Events) != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(stats.Countries) != 2 || stats.Countries[0].Value != "US" || stats.Countries[0].Count != 4 {
		t.Fatalf("unexpected countries: %+v", stats.Countries)
	}
	filtered, err := store.Stats(ctx, dao.AnalyticsCaptureFilter{
		OrgID: "org-1", SourceID: source.ID, Interval: "day",
		From: hour, To: hour.Add(2 * time.Hour),
		Event: "click", Countries: []string{"IN"}, Devices: []string{"mobile"},
	})
	if err != nil || filtered.Total != 3 {
		t.Fatalf("filtered stats: %+v, %v", filtered, err)
	}
	totals, err := store.SumBySources(ctx, dao.AnalyticsBulkFilter{
		OrgID: "org-1", SourceIDs: []string{source.ID}, From: hour, To: hour.Add(2 * time.Hour),
	})
	if err != nil || len(totals) != 1 || totals[0].Count != 7 {
		t.Fatalf("sum by sources: %+v, %v", totals, err)
	}

	disabled := true
	updated, err := store.Update(ctx, "org-1", source.ID, dao.AnalyticsSourceUpdate{
		Disabled: &disabled, UpdatedAt: now.Add(time.Minute),
	})
	if err != nil || !updated.Disabled() {
		t.Fatalf("disable source: %+v, %v", updated, err)
	}
	if err := store.Delete(ctx, "org-1", source.ID); err != nil {
		t.Fatalf("delete source: %v", err)
	}
	if _, err := store.FindByID(ctx, "org-1", source.ID); !errors.Is(err, entities.ErrAnalyticsSourceNotFound) {
		t.Fatalf("expected deleted source to be gone, got %v", err)
	}
}

func clearAnalyticsCollections(ctx context.Context, client *drivermongo.Client, database string) error {
	for _, collection := range []string{"analytics_sources", "analytics_captures_hourly"} {
		if _, err := client.Database(database).Collection(collection).DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
	}
	return nil
}
