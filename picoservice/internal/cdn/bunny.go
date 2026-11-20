// Package cdn integrates with the CDN used for public short-link redirects.
package cdn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBunnyPurgeEndpoint = "https://api.bunny.net/purge"

// BunnyConfig contains the Bunny purge client settings.
type BunnyConfig struct {
	APIKey   string
	Endpoint string
	Client   *http.Client
}

// BunnyPurger purges cached URLs through the Bunny.net API.
type BunnyPurger struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewBunnyPurger returns a Bunny cache purger. Endpoint and Client are optional
// overrides used by tests.
func NewBunnyPurger(cfg BunnyConfig) (*BunnyPurger, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("cdn: bunny api key is required")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultBunnyPurgeEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("cdn: bunny purge endpoint must be an absolute http or https URL")
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	return &BunnyPurger{apiKey: apiKey, endpoint: endpoint, client: client}, nil
}

// Purge removes target from the Bunny cache. The call waits for the purge to
// finish so a successful mutation is not followed by a stale cached redirect.
func (p *BunnyPurger) Purge(ctx context.Context, target string) error {
	targetURL, err := url.Parse(target)
	if err != nil || (targetURL.Scheme != "http" && targetURL.Scheme != "https") || targetURL.Host == "" {
		return fmt.Errorf("cdn: purge target must be an absolute http or https URL")
	}
	target = targetURL.String()

	endpoint, err := url.Parse(p.endpoint)
	if err != nil {
		return fmt.Errorf("cdn: parse purge endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("url", target)
	query.Set("async", "false")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("cdn: create purge request: %w", err)
	}
	request.Header.Set("AccessKey", p.apiKey)

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("cdn: purge request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("cdn: purge returned status %d", response.StatusCode)
	}
	return nil
}
