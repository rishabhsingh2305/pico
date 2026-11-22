// Package bunny implements the Bunny.net CDN Logging v2 client.
package bunny

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"picoservice/internal/analytics"
)

const (
	defaultEndpoint      = "https://logging.bunnycdn.com"
	defaultClientTimeout = 30 * time.Second
	defaultRetryDelay    = time.Second
	defaultMaxAttempts   = 5
	maxResponseBytes     = 32 << 20
)

// Config contains the Bunny Logging API settings.
type Config struct {
	APIKey      string
	PullZoneID  int64
	Endpoint    string
	Client      *http.Client
	RetryDelay  time.Duration
	MaxAttempts int
}

// Client fetches request logs from Bunny's Logging API v2.
type Client struct {
	apiKey      string
	pullZoneID  int64
	endpoint    string
	client      *http.Client
	retryDelay  time.Duration
	maxAttempts int
}

// New returns a Bunny log client. Endpoint, Client, RetryDelay, and MaxAttempts
// are optional overrides used by tests.
func New(cfg Config) (*Client, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("bunny logs: api key is required")
	}
	if cfg.PullZoneID <= 0 {
		return nil, fmt.Errorf("bunny logs: pull zone id must be positive")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("bunny logs: endpoint must be an absolute http or https URL")
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: defaultClientTimeout}
	}

	retryDelay := cfg.RetryDelay
	if retryDelay <= 0 {
		retryDelay = defaultRetryDelay
	}
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	return &Client{
		apiKey:      apiKey,
		pullZoneID:  cfg.PullZoneID,
		endpoint:    strings.TrimRight(endpoint, "/"),
		client:      client,
		retryDelay:  retryDelay,
		maxAttempts: maxAttempts,
	}, nil
}

// List returns one page of logs. It retries rate limits, server errors, and
// transient network failures.
func (c *Client) List(ctx context.Context, query analytics.LogQuery) (analytics.LogPage, error) {
	if query.From.IsZero() || query.To.IsZero() || !query.To.After(query.From) {
		return analytics.LogPage{}, fmt.Errorf("bunny logs: from and to must define a positive range")
	}
	if query.Limit < 1 || query.Limit > 10000 {
		return analytics.LogPage{}, fmt.Errorf("bunny logs: limit must be between 1 and 10000")
	}
	if query.Offset < 0 {
		return analytics.LogPage{}, fmt.Errorf("bunny logs: offset must be non-negative")
	}

	target, err := c.targetURL(query)
	if err != nil {
		return analytics.LogPage{}, err
	}

	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		page, retryable, retryAfter, err := c.fetch(ctx, target)
		if err == nil {
			page.NextOffset = query.Offset + len(page.Entries)
			return page, nil
		}
		lastErr = err
		if !retryable || attempt == c.maxAttempts {
			return analytics.LogPage{}, err
		}

		delay := c.retryDelay * time.Duration(attempt)
		if retryAfter > delay {
			delay = retryAfter
		}
		if err := wait(ctx, delay); err != nil {
			return analytics.LogPage{}, err
		}
	}
	return analytics.LogPage{}, lastErr
}

func (c *Client) targetURL(query analytics.LogQuery) (string, error) {
	parsed, err := url.Parse(c.endpoint)
	if err != nil {
		return "", fmt.Errorf("bunny logs: parse endpoint: %w", err)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + fmt.Sprintf("/v2/pullzones/%d/logs", c.pullZoneID)

	values := parsed.Query()
	values.Set("from", query.From.UTC().Format(time.RFC3339))
	values.Set("to", query.To.UTC().Format(time.RFC3339))
	// Bunny's v2 urlContains filter matches the request path, not host+path.
	// Use the path prefix here and enforce the host in LinkHourProcessor.
	contains := strings.TrimSpace(query.PathPrefix)
	if contains == "" {
		contains = strings.TrimSpace(query.Host)
	}
	if contains != "" {
		values.Set("urlContains", contains)
	}
	if query.Status != "" {
		values.Set("status", query.Status)
	}
	values.Set("order", "asc")
	values.Set("limit", strconv.Itoa(query.Limit))
	values.Set("offset", strconv.Itoa(query.Offset))
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func (c *Client) fetch(ctx context.Context, target string) (analytics.LogPage, bool, time.Duration, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return analytics.LogPage{}, false, 0, fmt.Errorf("bunny logs: create request: %w", err)
	}
	request.Header.Set("AccessKey", c.apiKey)
	request.Header.Set("Accept", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return analytics.LogPage{}, false, 0, ctx.Err()
		}
		return analytics.LogPage{}, true, 0, fmt.Errorf("bunny logs: request failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return analytics.LogPage{}, true, 0, fmt.Errorf("bunny logs: read response: %w", err)
	}

	if response.StatusCode == http.StatusOK {
		page, err := decodePage(body)
		if err != nil {
			return analytics.LogPage{}, false, 0, err
		}
		return page, false, 0, nil
	}

	retryAfter := parseRetryAfter(response.Header.Get("Retry-After"))
	apiErr := decodeError(response.StatusCode, body)
	switch {
	case response.StatusCode == http.StatusTooManyRequests:
		return analytics.LogPage{}, true, retryAfter, apiErr
	case response.StatusCode >= 500:
		return analytics.LogPage{}, true, retryAfter, apiErr
	default:
		return analytics.LogPage{}, false, 0, apiErr
	}
}

func decodePage(body []byte) (analytics.LogPage, error) {
	var payload struct {
		Data []struct {
			Timestamp   string  `json:"timestamp"`
			RequestID   *string `json:"requestId"`
			Host        *string `json:"host"`
			Path        *string `json:"path"`
			StatusCode  int     `json:"statusCode"`
			CacheStatus *string `json:"cacheStatus"`
			RemoteIP    *string `json:"remoteIp"`
			UserAgent   *string `json:"userAgent"`
		} `json:"data"`
		Pagination struct {
			HasMore bool `json:"hasMore"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return analytics.LogPage{}, fmt.Errorf("bunny logs: decode response: %w", err)
	}

	entries := make([]analytics.LogEntry, 0, len(payload.Data))
	for _, item := range payload.Data {
		timestamp, err := time.Parse(time.RFC3339Nano, item.Timestamp)
		if err != nil {
			return analytics.LogPage{}, fmt.Errorf("bunny logs: parse timestamp %q: %w", item.Timestamp, err)
		}
		entries = append(entries, analytics.LogEntry{
			Timestamp:   timestamp,
			RequestID:   stringValue(item.RequestID),
			Host:        stringValue(item.Host),
			Path:        stringValue(item.Path),
			StatusCode:  item.StatusCode,
			CacheStatus: stringValue(item.CacheStatus),
			RemoteIP:    stringValue(item.RemoteIP),
			UserAgent:   stringValue(item.UserAgent),
		})
	}

	return analytics.LogPage{
		Entries: entries,
		HasMore: payload.Pagination.HasMore && len(entries) > 0,
	}, nil
}

func decodeError(status int, body []byte) error {
	var payload struct {
		Error struct {
			Code    string   `json:"code"`
			Message string   `json:"message"`
			Details []string `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)

	message := strings.TrimSpace(payload.Error.Message)
	if message == "" {
		message = http.StatusText(status)
	}
	if message == "" {
		message = "unexpected response"
	}
	return &Error{
		StatusCode: status,
		Code:       payload.Error.Code,
		Message:    message,
		Details:    payload.Error.Details,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseRetryAfter(raw string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Error is a structured Bunny Logging API error.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	Details    []string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("bunny logs: status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("bunny logs: status %d (%s): %s", e.StatusCode, e.Code, e.Message)
}
