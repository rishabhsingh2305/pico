package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"picoservice/internal/analytics/bunny"
	"picoservice/internal/dao/mongodb"
	"picoservice/internal/services"
)

const bunnyPullZoneIDEnv = "BUNNY_PULL_ZONE_ID"

// AnalyticsConfig contains the link analytics job settings.
type AnalyticsConfig struct {
	BunnyAPIKey     string
	BunnyPullZoneID int64
	ShortLinkHost   string
	PageSize        int
	Job             services.LinkAnalyticsJobConfig
}

// AnalyticsConfigFromEnv reads the link analytics job configuration.
func AnalyticsConfigFromEnv() (AnalyticsConfig, error) {
	pullZoneRaw := strings.TrimSpace(os.Getenv(bunnyPullZoneIDEnv))
	if pullZoneRaw == "" {
		return AnalyticsConfig{}, fmt.Errorf("%s is not set", bunnyPullZoneIDEnv)
	}
	pullZoneID, err := strconv.ParseInt(pullZoneRaw, 10, 64)
	if err != nil || pullZoneID <= 0 {
		return AnalyticsConfig{}, fmt.Errorf("%s must be a positive integer", bunnyPullZoneIDEnv)
	}

	apiKey := strings.TrimSpace(os.Getenv(bunnyAPIKeyEnv))
	if apiKey == "" {
		return AnalyticsConfig{}, fmt.Errorf("%s is not set", bunnyAPIKeyEnv)
	}

	host, err := shortLinkHost(os.Getenv(shortBaseURLEnv))
	if err != nil {
		return AnalyticsConfig{}, err
	}

	return AnalyticsConfig{
		BunnyAPIKey:     apiKey,
		BunnyPullZoneID: pullZoneID,
		ShortLinkHost:   host,
		PageSize:        services.DefaultLinkAnalyticsPageSize,
		Job: services.LinkAnalyticsJobConfig{
			PullZoneID:      pullZoneID,
			LeaseDuration:   services.DefaultLinkAnalyticsLeaseDuration,
			SafetyLag:       services.DefaultLinkAnalyticsSafetyLag,
			MaxHoursPerRun:  services.DefaultLinkAnalyticsMaxHoursPerRun,
			InitialBackfill: services.DefaultLinkAnalyticsInitialBackfill,
			MaxLookback:     services.DefaultLinkAnalyticsMaxLookback,
		},
	}, nil
}

func shortLinkHost(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return "", fmt.Errorf("%s must be an absolute http or https URL", shortBaseURLEnv)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return "", fmt.Errorf("%s must not contain a path, query, or fragment", shortBaseURLEnv)
	}
	return strings.ToLower(parsed.Hostname()), nil
}

// NewLinkAnalyticsJob wires the Bunny log client, link resolver, click store,
// and job state store into an ingester.
func NewLinkAnalyticsJob(db *mongodb.Client, cfg AnalyticsConfig) (*services.LinkAnalyticsIngester, error) {
	if db == nil {
		return nil, errors.New("app: mongodb client is required")
	}
	if cfg.BunnyAPIKey == "" || cfg.BunnyPullZoneID <= 0 || cfg.ShortLinkHost == "" {
		return nil, errors.New("app: incomplete analytics config")
	}
	if cfg.PageSize <= 0 {
		return nil, errors.New("app: analytics page size must be positive")
	}

	logClient, err := bunny.New(bunny.Config{
		APIKey:     cfg.BunnyAPIKey,
		PullZoneID: cfg.BunnyPullZoneID,
	})
	if err != nil {
		return nil, fmt.Errorf("app: create bunny log client: %w", err)
	}

	linkStore := mongodb.NewLinkStore(db)
	linkAnalyticsStore := mongodb.NewLinkAnalyticsStore(db)
	analyticsStore := mongodb.NewAnalyticsStore(db)
	linkProcessor := services.NewLinkHourProcessor(logClient, linkStore, cfg.ShortLinkHost, cfg.PageSize)
	captureProcessor := services.NewAnalyticsCaptureProcessor(logClient, analyticsStore, cfg.ShortLinkHost, cfg.PageSize)

	ingester := services.NewLinkAnalyticsIngester(
		linkProcessor,
		linkAnalyticsStore,
		linkAnalyticsStore,
		cfg.Job,
		time.Now,
	)
	ingester.SetCapturePipeline(services.AnalyticsCapturePipeline{
		Processor: captureProcessor,
		Writer:    analyticsStore,
	})
	return ingester, nil
}
