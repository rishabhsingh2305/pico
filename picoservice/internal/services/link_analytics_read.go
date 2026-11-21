package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/validation"
)

const (
	// LinkStatsMaxIDs is the maximum number of links accepted by one stats read.
	LinkStatsMaxIDs = 100
	// LinkStatsMaxRange is the maximum date range accepted by one stats read.
	LinkStatsMaxRange = 365 * 24 * time.Hour
)

// LinkStatsInput contains the link analytics read filters.
type LinkStatsInput struct {
	OrgID   string
	LinkIDs []string
	From    time.Time
	To      time.Time
}

// LinkStat is one link's click total.
type LinkStat struct {
	LinkID string
	Clicks int64
}

// LinkStats is a collection of link click totals.
type LinkStats struct {
	Items []LinkStat
}

// LinkStatsReader reads link click totals.
type LinkStatsReader struct {
	store dao.LinkClickReader
}

// NewLinkStatsReader returns a reader backed by store.
func NewLinkStatsReader(store dao.LinkClickReader) *LinkStatsReader {
	return &LinkStatsReader{store: store}
}

// Stats returns click totals for the requested links and range.
func (r *LinkStatsReader) Stats(ctx context.Context, in LinkStatsInput) (LinkStats, error) {
	if r.store == nil {
		return LinkStats{}, errors.New("link stats reader: store is required")
	}
	if in.OrgID == "" {
		return LinkStats{}, &validation.Error{Field: "orgId", Message: "orgId is required"}
	}

	linkIDs := uniqueSortedIDs(in.LinkIDs)
	if len(linkIDs) == 0 {
		return LinkStats{}, &validation.Error{Field: "ids", Message: "ids is required"}
	}
	if len(linkIDs) > LinkStatsMaxIDs {
		return LinkStats{}, &validation.Error{
			Field:   "ids",
			Message: fmt.Sprintf("ids must contain at most %d items", LinkStatsMaxIDs),
		}
	}
	if in.From.IsZero() || in.To.IsZero() {
		return LinkStats{}, &validation.Error{Field: "from", Message: "from and to are required"}
	}
	if !in.From.Before(in.To) {
		return LinkStats{}, &validation.Error{Field: "from", Message: "from must be before to"}
	}
	if in.To.Sub(in.From) > LinkStatsMaxRange {
		return LinkStats{}, &validation.Error{
			Field:   "to",
			Message: fmt.Sprintf("range must be at most %d days", int(LinkStatsMaxRange/(24*time.Hour))),
		}
	}

	totals, err := r.store.SumByLinks(ctx, dao.LinkStatsFilter{
		OrgID:   in.OrgID,
		LinkIDs: linkIDs,
		From:    in.From.UTC(),
		To:      in.To.UTC(),
	})
	if err != nil {
		return LinkStats{}, fmt.Errorf("read link stats: %w", err)
	}

	items := make([]LinkStat, 0, len(totals))
	for _, total := range totals {
		items = append(items, LinkStat{
			LinkID: total.LinkID,
			Clicks: total.Clicks,
		})
	}
	return LinkStats{Items: items}, nil
}

func uniqueSortedIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	sort.Strings(unique)
	return unique
}
