package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"picoservice/internal/analytics"
	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

// AnalyticsSourceResolver resolves public pixel keys to sources.
type AnalyticsSourceResolver interface {
	FindByKeys(ctx context.Context, keys []string) ([]entities.AnalyticsSource, error)
}

// AnalyticsCaptureProcessor turns one time range of /p/ logs into capture
// counters. It does not write to the database or manage job state.
type AnalyticsCaptureProcessor struct {
	logs     analytics.LogSource
	sources  AnalyticsSourceResolver
	host     string
	pageSize int
}

// NewAnalyticsCaptureProcessor returns a processor for one CDN host.
func NewAnalyticsCaptureProcessor(logs analytics.LogSource, sources AnalyticsSourceResolver, host string, pageSize int) *AnalyticsCaptureProcessor {
	if pageSize <= 0 {
		pageSize = DefaultLinkAnalyticsPageSize
	}
	return &AnalyticsCaptureProcessor{
		logs:     logs,
		sources:  sources,
		host:     analytics.NormalizeHost(host),
		pageSize: pageSize,
	}
}

// Process returns capture counters for one time range.
func (p *AnalyticsCaptureProcessor) Process(ctx context.Context, from, to time.Time) ([]dao.AnalyticsCaptureCount, error) {
	if p.logs == nil {
		return nil, errors.New("analytics capture processor: log source is required")
	}
	if p.sources == nil {
		return nil, errors.New("analytics capture processor: source resolver is required")
	}
	if p.host == "" {
		return nil, errors.New("analytics capture processor: host is required")
	}
	from = from.UTC()
	to = to.UTC()
	if !to.After(from) {
		return nil, errors.New("analytics capture processor: to must be after from")
	}

	type captureKey struct {
		sourceKey    string
		event        string
		countryCode  string
		device       string
		isBot        bool
		referrerHost string
	}

	seen := make(map[string]struct{})
	counts := make(map[captureKey]int64)
	offset := 0
	for {
		page, err := p.logs.List(ctx, analytics.LogQuery{
			From:       from,
			To:         to,
			Host:       p.host,
			PathPrefix: "/p/",
			Status:     "2xx,3xx",
			Limit:      p.pageSize,
			Offset:     offset,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch capture logs: %w", err)
		}

		for _, entry := range page.Entries {
			if analytics.NormalizeHost(entry.Host) != p.host {
				continue
			}
			if entry.StatusCode < 200 || entry.StatusCode >= 400 {
				continue
			}
			key := analytics.RequestKey(entry)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			sourceKey, event, ok := parseCapturePath(entry.Path)
			if !ok {
				continue
			}
			isBot := analyticsIsBot(entry.UserAgent)
			counts[captureKey{
				sourceKey:    sourceKey,
				event:        event,
				countryCode:  analyticsCountryCode(entry.CountryCode),
				device:       analyticsDevice(entry.UserAgent),
				isBot:        isBot,
				referrerHost: analyticsReferrerHost(entry.Referer),
			}]++
		}

		if !page.HasMore {
			break
		}
		if page.NextOffset <= offset {
			return nil, errors.New("analytics capture processor: log pagination did not advance")
		}
		offset = page.NextOffset
	}

	if len(counts) == 0 {
		return []dao.AnalyticsCaptureCount{}, nil
	}

	sourceKeys := make([]string, 0, len(counts))
	seenKeys := make(map[string]struct{}, len(counts))
	for key := range counts {
		if _, ok := seenKeys[key.sourceKey]; ok {
			continue
		}
		seenKeys[key.sourceKey] = struct{}{}
		sourceKeys = append(sourceKeys, key.sourceKey)
	}
	sourcesByKey, err := p.resolveSources(ctx, sourceKeys)
	if err != nil {
		return nil, err
	}

	items := make([]dao.AnalyticsCaptureCount, 0, len(counts))
	for key, count := range counts {
		source, ok := sourcesByKey[key.sourceKey]
		if !ok || source.Disabled() || !source.AllowsEvent(key.event) {
			continue
		}
		items = append(items, dao.AnalyticsCaptureCount{
			OrgID:        source.OrgID,
			SourceID:     source.ID,
			Event:        key.event,
			HourStart:    from,
			CountryCode:  key.countryCode,
			Device:       key.device,
			IsBot:        key.isBot,
			ReferrerHost: key.referrerHost,
			Count:        count,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SourceID != items[j].SourceID {
			return items[i].SourceID < items[j].SourceID
		}
		if items[i].Event != items[j].Event {
			return items[i].Event < items[j].Event
		}
		if items[i].CountryCode != items[j].CountryCode {
			return items[i].CountryCode < items[j].CountryCode
		}
		if items[i].Device != items[j].Device {
			return items[i].Device < items[j].Device
		}
		if items[i].IsBot != items[j].IsBot {
			return !items[i].IsBot
		}
		return items[i].ReferrerHost < items[j].ReferrerHost
	})
	return items, nil
}

func (p *AnalyticsCaptureProcessor) resolveSources(ctx context.Context, keys []string) (map[string]entities.AnalyticsSource, error) {
	sort.Strings(keys)
	byKey := make(map[string]entities.AnalyticsSource, len(keys))
	for start := 0; start < len(keys); start += 500 {
		end := start + 500
		if end > len(keys) {
			end = len(keys)
		}
		sources, err := p.sources.FindByKeys(ctx, keys[start:end])
		if err != nil {
			return nil, fmt.Errorf("resolve analytics sources: %w", err)
		}
		for _, source := range sources {
			byKey[source.Key] = source
		}
	}
	return byKey, nil
}

func parseCapturePath(raw string) (string, string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", false
	}
	path := parsed.Path
	if !strings.HasPrefix(path, "/p/") {
		return "", "", false
	}
	key := strings.TrimPrefix(path, "/p/")
	if key == "" || strings.Contains(key, "/") {
		return "", "", false
	}
	event := "view"
	if rawEvent := parsed.Query().Get("event"); rawEvent != "" {
		normalized, ok := normalizeAnalyticsEvent(rawEvent)
		if !ok {
			return "", "", false
		}
		event = normalized
	}
	return key, event, true
}
