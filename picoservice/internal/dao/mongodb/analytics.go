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

const (
	analyticsSourcesCollection  = "analytics_sources"
	analyticsCapturesCollection = "analytics_captures_hourly"
)

// AnalyticsStore persists analytics sources and capture counters.
type AnalyticsStore struct {
	sources  *mongo.Collection
	captures *mongo.Collection
}

var (
	_ dao.AnalyticsSourceDAO     = (*AnalyticsStore)(nil)
	_ dao.AnalyticsCaptureWriter = (*AnalyticsStore)(nil)
	_ dao.AnalyticsCaptureReader = (*AnalyticsStore)(nil)
)

// NewAnalyticsStore returns an analytics store backed by client.
func NewAnalyticsStore(client *Client) *AnalyticsStore {
	return &AnalyticsStore{
		sources:  client.collection(analyticsSourcesCollection),
		captures: client.collection(analyticsCapturesCollection),
	}
}

type analyticsSourceDocument struct {
	ID          bson.ObjectID `bson:"_id"`
	OrgID       string        `bson:"orgId"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Key         string        `bson:"key"`
	Events      []string      `bson:"events"`
	DisabledAt  *time.Time    `bson:"disabledAt,omitempty"`
	CreatedAt   time.Time     `bson:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt"`
}

func (d analyticsSourceDocument) source() entities.AnalyticsSource {
	events := d.Events
	if events == nil {
		events = []string{}
	}
	return entities.AnalyticsSource{
		ID:          d.ID.Hex(),
		OrgID:       d.OrgID,
		Name:        d.Name,
		Description: d.Description,
		Key:         d.Key,
		Events:      events,
		DisabledAt:  d.DisabledAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// Insert stores an analytics source. A duplicate key maps to ErrAnalyticsKeyTaken.
func (s *AnalyticsStore) Insert(ctx context.Context, source entities.AnalyticsSource) (entities.AnalyticsSource, error) {
	events := source.Events
	if events == nil {
		events = []string{}
	}
	document := analyticsSourceDocument{
		ID:          bson.NewObjectID(),
		OrgID:       source.OrgID,
		Name:        source.Name,
		Description: source.Description,
		Key:         source.Key,
		Events:      events,
		DisabledAt:  source.DisabledAt,
		CreatedAt:   source.CreatedAt,
		UpdatedAt:   source.UpdatedAt,
	}
	if _, err := s.sources.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return entities.AnalyticsSource{}, entities.ErrAnalyticsKeyTaken
		}
		return entities.AnalyticsSource{}, fmt.Errorf("insert analytics source: %w", err)
	}
	return document.source(), nil
}

// FindByID returns an analytics source scoped to orgID.
func (s *AnalyticsStore) FindByID(ctx context.Context, orgID, id string) (entities.AnalyticsSource, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
	}
	var document analyticsSourceDocument
	err = s.sources.FindOne(ctx, bson.M{"_id": objectID, "orgId": orgID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
	}
	if err != nil {
		return entities.AnalyticsSource{}, fmt.Errorf("find analytics source: %w", err)
	}
	return document.source(), nil
}

// FindByKey returns an analytics source by its public pixel key.
func (s *AnalyticsStore) FindByKey(ctx context.Context, key string) (entities.AnalyticsSource, error) {
	var document analyticsSourceDocument
	err := s.sources.FindOne(ctx, bson.M{"key": key}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
	}
	if err != nil {
		return entities.AnalyticsSource{}, fmt.Errorf("find analytics source by key: %w", err)
	}
	return document.source(), nil
}

// FindByKeys returns sources for a batch of pixel keys.
func (s *AnalyticsStore) FindByKeys(ctx context.Context, keys []string) ([]entities.AnalyticsSource, error) {
	if len(keys) == 0 {
		return []entities.AnalyticsSource{}, nil
	}
	cursor, err := s.sources.Find(ctx, bson.M{"key": bson.M{"$in": keys}})
	if err != nil {
		return nil, fmt.Errorf("find analytics sources by keys: %w", err)
	}
	var documents []analyticsSourceDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode analytics sources: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close analytics source cursor: %w", err)
	}
	items := make([]entities.AnalyticsSource, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.source())
	}
	return items, nil
}

// List returns one page of analytics sources for an organization.
func (s *AnalyticsStore) List(ctx context.Context, filter dao.AnalyticsSourceFilter) ([]entities.AnalyticsSource, error) {
	query := bson.M{"orgId": filter.OrgID}
	if filter.ActiveOnly {
		query["disabledAt"] = nil
	}
	if filter.Query != "" {
		query["name"] = bson.M{"$regex": "(?i)^" + regexp.QuoteMeta(filter.Query)}
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(filter.Limit)).
		SetSkip(int64(filter.Offset))
	cursor, err := s.sources.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("list analytics sources: %w", err)
	}
	var documents []analyticsSourceDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode analytics source list: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close analytics source list cursor: %w", err)
	}
	items := make([]entities.AnalyticsSource, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.source())
	}
	return items, nil
}

// Update applies optional fields to an analytics source.
func (s *AnalyticsStore) Update(ctx context.Context, orgID, id string, update dao.AnalyticsSourceUpdate) (entities.AnalyticsSource, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
	}
	set := bson.M{"updatedAt": update.UpdatedAt}
	unset := bson.M{}
	if update.Name != nil {
		set["name"] = *update.Name
	}
	if update.Description != nil {
		set["description"] = *update.Description
	}
	if update.Events != nil {
		set["events"] = *update.Events
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
	var document analyticsSourceDocument
	err = s.sources.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID},
		updateDocument,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.AnalyticsSource{}, entities.ErrAnalyticsSourceNotFound
	}
	if err != nil {
		return entities.AnalyticsSource{}, fmt.Errorf("update analytics source: %w", err)
	}
	return document.source(), nil
}

// Delete removes an analytics source. Historical capture counters remain.
func (s *AnalyticsStore) Delete(ctx context.Context, orgID, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrAnalyticsSourceNotFound
	}
	result, err := s.sources.DeleteOne(ctx, bson.M{"_id": objectID, "orgId": orgID})
	if err != nil {
		return fmt.Errorf("delete analytics source: %w", err)
	}
	if result.DeletedCount == 0 {
		return entities.ErrAnalyticsSourceNotFound
	}
	return nil
}

type analyticsCaptureDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	OrgID        string        `bson:"orgId"`
	SourceID     string        `bson:"sourceId"`
	Event        string        `bson:"event"`
	HourStart    time.Time     `bson:"hourStart"`
	CountryCode  string        `bson:"countryCode"`
	Device       string        `bson:"device"`
	IsBot        bool          `bson:"isBot"`
	ReferrerHost string        `bson:"referrerHost"`
	Count        int64         `bson:"count"`
	UpdatedAt    time.Time     `bson:"updatedAt"`
}

// UpsertHourly stores monotonic capture counts at the dimension grain.
func (s *AnalyticsStore) UpsertHourly(ctx context.Context, counts []dao.AnalyticsCaptureCount) error {
	if len(counts) == 0 {
		return nil
	}
	now := time.Now().UTC()
	models := make([]mongo.WriteModel, 0, len(counts))
	for _, count := range counts {
		if count.Count <= 0 {
			continue
		}
		if count.OrgID == "" || count.SourceID == "" || count.Event == "" || count.HourStart.IsZero() {
			return errors.New("upsert analytics captures: org, source, event, and hour are required")
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{
				"sourceId":     count.SourceID,
				"event":        count.Event,
				"hourStart":    count.HourStart.UTC(),
				"countryCode":  count.CountryCode,
				"device":       count.Device,
				"isBot":        count.IsBot,
				"referrerHost": count.ReferrerHost,
			}).
			SetUpdate(bson.M{
				"$max": bson.M{"count": count.Count},
				"$set": bson.M{
					"orgId":     count.OrgID,
					"updatedAt": now,
				},
			}).
			SetUpsert(true))
	}
	if len(models) == 0 {
		return nil
	}
	if _, err := s.captures.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return fmt.Errorf("upsert analytics captures: %w", err)
	}
	return nil
}

// Stats returns totals, series, and breakdowns for one source.
func (s *AnalyticsStore) Stats(ctx context.Context, filter dao.AnalyticsCaptureFilter) (dao.AnalyticsStats, error) {
	interval := strings.TrimSpace(filter.Interval)
	if interval != "hour" && interval != "day" {
		return dao.AnalyticsStats{}, fmt.Errorf("analytics stats: invalid interval %q", interval)
	}
	match := analyticsCaptureMatch(filter)
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$facet", Value: bson.M{
			"total": bson.A{
				bson.D{{Key: "$group", Value: bson.M{"_id": nil, "count": bson.M{"$sum": "$count"}}}},
			},
			"series": bson.A{
				bson.D{{Key: "$group", Value: bson.M{
					"_id":   bson.M{"$dateTrunc": bson.M{"date": "$hourStart", "unit": interval, "timezone": "UTC"}},
					"count": bson.M{"$sum": "$count"},
				}}},
				bson.D{{Key: "$sort", Value: bson.M{"_id": 1}}},
			},
			"events": bson.A{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$event", "count": bson.M{"$sum": "$count"}}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1, "_id": 1}}},
			},
			"countries": bson.A{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$countryCode", "count": bson.M{"$sum": "$count"}}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1, "_id": 1}}},
			},
			"devices": bson.A{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$device", "count": bson.M{"$sum": "$count"}}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1, "_id": 1}}},
			},
			"bots": bson.A{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$isBot", "count": bson.M{"$sum": "$count"}}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1, "_id": 1}}},
			},
			"referrers": bson.A{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$referrerHost", "count": bson.M{"$sum": "$count"}}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1, "_id": 1}}},
			},
		}}},
	}
	cursor, err := s.captures.Aggregate(ctx, pipeline)
	if err != nil {
		return dao.AnalyticsStats{}, fmt.Errorf("aggregate analytics stats: %w", err)
	}
	var result []struct {
		Total []struct {
			Count int64 `bson:"count"`
		} `bson:"total"`
		Series []struct {
			Bucket time.Time `bson:"_id"`
			Count  int64     `bson:"count"`
		} `bson:"series"`
		Events []struct {
			Event string `bson:"_id"`
			Count int64  `bson:"count"`
		} `bson:"events"`
		Countries []struct {
			Value string `bson:"_id"`
			Count int64  `bson:"count"`
		} `bson:"countries"`
		Devices []struct {
			Value string `bson:"_id"`
			Count int64  `bson:"count"`
		} `bson:"devices"`
		Bots []struct {
			Value bool  `bson:"_id"`
			Count int64 `bson:"count"`
		} `bson:"bots"`
		Referrers []struct {
			Value string `bson:"_id"`
			Count int64  `bson:"count"`
		} `bson:"referrers"`
	}
	if err := cursor.All(ctx, &result); err != nil {
		_ = cursor.Close(ctx)
		return dao.AnalyticsStats{}, fmt.Errorf("decode analytics stats: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return dao.AnalyticsStats{}, fmt.Errorf("close analytics stats cursor: %w", err)
	}
	if len(result) == 0 {
		return dao.AnalyticsStats{
			Series:    []dao.AnalyticsSeriesPoint{},
			Events:    []dao.AnalyticsEventTotal{},
			Countries: []dao.AnalyticsBreakdown{},
			Devices:   []dao.AnalyticsBreakdown{},
			Bots:      []dao.AnalyticsBreakdown{},
			Referrers: []dao.AnalyticsBreakdown{},
		}, nil
	}
	facet := result[0]
	stats := dao.AnalyticsStats{
		Series:    make([]dao.AnalyticsSeriesPoint, 0, len(facet.Series)),
		Events:    make([]dao.AnalyticsEventTotal, 0, len(facet.Events)),
		Countries: make([]dao.AnalyticsBreakdown, 0, len(facet.Countries)),
		Devices:   make([]dao.AnalyticsBreakdown, 0, len(facet.Devices)),
		Bots:      make([]dao.AnalyticsBreakdown, 0, len(facet.Bots)),
		Referrers: make([]dao.AnalyticsBreakdown, 0, len(facet.Referrers)),
	}
	if len(facet.Total) > 0 {
		stats.Total = facet.Total[0].Count
	}
	for _, item := range facet.Series {
		stats.Series = append(stats.Series, dao.AnalyticsSeriesPoint{Bucket: item.Bucket, Count: item.Count})
	}
	for _, item := range facet.Events {
		stats.Events = append(stats.Events, dao.AnalyticsEventTotal{Event: item.Event, Count: item.Count})
	}
	for _, item := range facet.Countries {
		stats.Countries = append(stats.Countries, dao.AnalyticsBreakdown{Value: item.Value, Count: item.Count})
	}
	for _, item := range facet.Devices {
		stats.Devices = append(stats.Devices, dao.AnalyticsBreakdown{Value: item.Value, Count: item.Count})
	}
	for _, item := range facet.Bots {
		stats.Bots = append(stats.Bots, dao.AnalyticsBreakdown{Value: fmt.Sprint(item.Value), Count: item.Count})
	}
	for _, item := range facet.Referrers {
		stats.Referrers = append(stats.Referrers, dao.AnalyticsBreakdown{Value: item.Value, Count: item.Count})
	}
	return stats, nil
}

// SumBySources returns capture totals grouped by source.
func (s *AnalyticsStore) SumBySources(ctx context.Context, filter dao.AnalyticsBulkFilter) ([]dao.AnalyticsSourceTotal, error) {
	if len(filter.SourceIDs) == 0 {
		return []dao.AnalyticsSourceTotal{}, nil
	}
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"orgId":     filter.OrgID,
			"sourceId":  bson.M{"$in": filter.SourceIDs},
			"hourStart": bson.M{"$gte": filter.From.UTC(), "$lt": filter.To.UTC()},
		}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": "$sourceId", "count": bson.M{"$sum": "$count"}}}},
		bson.D{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}
	cursor, err := s.captures.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("sum analytics captures: %w", err)
	}
	var results []struct {
		SourceID string `bson:"_id"`
		Count    int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode analytics capture totals: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close analytics capture totals cursor: %w", err)
	}
	totals := make([]dao.AnalyticsSourceTotal, 0, len(results))
	for _, result := range results {
		totals = append(totals, dao.AnalyticsSourceTotal{SourceID: result.SourceID, Count: result.Count})
	}
	return totals, nil
}

func analyticsCaptureMatch(filter dao.AnalyticsCaptureFilter) bson.M {
	match := bson.M{
		"orgId":     filter.OrgID,
		"sourceId":  filter.SourceID,
		"hourStart": bson.M{"$gte": filter.From.UTC(), "$lt": filter.To.UTC()},
	}
	if filter.Event != "" {
		match["event"] = filter.Event
	}
	if len(filter.Countries) > 0 {
		match["countryCode"] = bson.M{"$in": filter.Countries}
	}
	if len(filter.Devices) > 0 {
		match["device"] = bson.M{"$in": filter.Devices}
	}
	if filter.Bot != nil {
		match["isBot"] = *filter.Bot
	}
	if filter.Referrer != "" {
		match["referrerHost"] = filter.Referrer
	}
	return match
}
