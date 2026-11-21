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
)

const (
	linkClicksCollection         = "link_clicks_hourly"
	linkAnalyticsStateCollection = "link_analytics_state"
)

// LinkAnalyticsStore persists link click aggregates and job state.
type LinkAnalyticsStore struct {
	clicks *mongo.Collection
	state  *mongo.Collection
}

var (
	_ dao.LinkClickWriter         = (*LinkAnalyticsStore)(nil)
	_ dao.LinkClickReader         = (*LinkAnalyticsStore)(nil)
	_ dao.LinkAnalyticsStateStore = (*LinkAnalyticsStore)(nil)
)

// NewLinkAnalyticsStore returns a link analytics store backed by client.
func NewLinkAnalyticsStore(client *Client) *LinkAnalyticsStore {
	return &LinkAnalyticsStore{
		clicks: client.collection(linkClicksCollection),
		state:  client.collection(linkAnalyticsStateCollection),
	}
}

type linkClickDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	OrgID     string        `bson:"orgId"`
	LinkID    string        `bson:"linkId"`
	HourStart time.Time     `bson:"hourStart"`
	Count     int64         `bson:"count"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// UpsertHourly stores monotonic hourly click counts. Zero counts are skipped.
func (s *LinkAnalyticsStore) UpsertHourly(ctx context.Context, counts []dao.LinkClickCount) error {
	if len(counts) == 0 {
		return nil
	}

	now := time.Now().UTC()
	models := make([]mongo.WriteModel, 0, len(counts))
	for _, count := range counts {
		if count.Count <= 0 {
			continue
		}
		if count.OrgID == "" || count.LinkID == "" || count.HourStart.IsZero() {
			return fmt.Errorf("upsert link clicks: org, link, and hour are required")
		}

		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{
				"linkId":    count.LinkID,
				"hourStart": count.HourStart.UTC(),
			}).
			SetUpdate(bson.M{
				"$max": bson.M{"count": count.Count},
				"$set": bson.M{
					"orgId":     count.OrgID,
					"updatedAt": now,
				},
				"$setOnInsert": bson.M{
					"linkId":    count.LinkID,
					"hourStart": count.HourStart.UTC(),
				},
			}).
			SetUpsert(true))
	}
	if len(models) == 0 {
		return nil
	}

	if _, err := s.clicks.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return fmt.Errorf("upsert link clicks: %w", err)
	}
	return nil
}

// SumByLinks returns click totals grouped by link for the requested range.
func (s *LinkAnalyticsStore) SumByLinks(ctx context.Context, filter dao.LinkStatsFilter) ([]dao.LinkClickTotal, error) {
	if len(filter.LinkIDs) == 0 {
		return []dao.LinkClickTotal{}, nil
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"orgId": filter.OrgID,
			"linkId": bson.M{
				"$in": filter.LinkIDs,
			},
			"hourStart": bson.M{
				"$gte": filter.From.UTC(),
				"$lt":  filter.To.UTC(),
			},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":    "$linkId",
			"clicks": bson.M{"$sum": "$count"},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}

	cursor, err := s.clicks.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("sum link clicks: %w", err)
	}

	var results []struct {
		LinkID string `bson:"_id"`
		Clicks int64  `bson:"clicks"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode link click totals: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close link click cursor: %w", err)
	}

	totals := make([]dao.LinkClickTotal, 0, len(results))
	for _, result := range results {
		totals = append(totals, dao.LinkClickTotal{
			LinkID: result.LinkID,
			Clicks: result.Clicks,
		})
	}
	return totals, nil
}

type linkAnalyticsStateDocument struct {
	PullZoneID        int64      `bson:"pullZoneId"`
	LastCompletedHour time.Time  `bson:"lastCompletedHour"`
	LockedUntil       *time.Time `bson:"lockedUntil,omitempty"`
	UpdatedAt         time.Time  `bson:"updatedAt"`
}

func (d linkAnalyticsStateDocument) state() dao.LinkAnalyticsState {
	return dao.LinkAnalyticsState{
		PullZoneID:        d.PullZoneID,
		LastCompletedHour: d.LastCompletedHour,
		LockedUntil:       d.LockedUntil,
		UpdatedAt:         d.UpdatedAt,
	}
}

// Acquire takes the job lease for pullZoneID. It returns acquired=false when
// another run still holds the lease.
func (s *LinkAnalyticsStore) Acquire(
	ctx context.Context,
	pullZoneID int64,
	now, leaseUntil, initialHour time.Time,
) (dao.LinkAnalyticsState, bool, error) {
	filter := bson.M{
		"pullZoneId": pullZoneID,
		"$or": bson.A{
			bson.M{"lockedUntil": nil},
			bson.M{"lockedUntil": bson.M{"$lte": now.UTC()}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"lockedUntil": leaseUntil.UTC(),
			"updatedAt":   now.UTC(),
		},
		"$setOnInsert": bson.M{
			"pullZoneId":        pullZoneID,
			"lastCompletedHour": initialHour.UTC(),
		},
	}

	var document linkAnalyticsStateDocument
	err := s.state.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().
			SetUpsert(true).
			SetReturnDocument(options.After),
	).Decode(&document)
	if err == nil {
		return document.state(), true, nil
	}
	if mongo.IsDuplicateKeyError(err) || errors.Is(err, mongo.ErrNoDocuments) {
		return dao.LinkAnalyticsState{}, false, nil
	}
	return dao.LinkAnalyticsState{}, false, fmt.Errorf("acquire link analytics lease: %w", err)
}

// Advance moves the completed-hour watermark forward without ever moving it
// backward.
func (s *LinkAnalyticsStore) Advance(ctx context.Context, pullZoneID int64, hour time.Time) error {
	_, err := s.state.UpdateOne(
		ctx,
		bson.M{"pullZoneId": pullZoneID},
		bson.M{
			"$max": bson.M{"lastCompletedHour": hour.UTC()},
			"$set": bson.M{"updatedAt": time.Now().UTC()},
		},
	)
	if err != nil {
		return fmt.Errorf("advance link analytics state: %w", err)
	}
	return nil
}

// Release clears the job lease.
func (s *LinkAnalyticsStore) Release(ctx context.Context, pullZoneID int64, now time.Time) error {
	_, err := s.state.UpdateOne(
		ctx,
		bson.M{"pullZoneId": pullZoneID},
		bson.M{
			"$unset": bson.M{"lockedUntil": ""},
			"$set":   bson.M{"updatedAt": now.UTC()},
		},
	)
	if err != nil {
		return fmt.Errorf("release link analytics lease: %w", err)
	}
	return nil
}
