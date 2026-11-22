// Package analytics contains pure request-log types and parsing helpers. It has
// no HTTP, database, or AWS dependencies.
package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// LogEntry is one request record returned by the CDN log API.
type LogEntry struct {
	Timestamp   time.Time
	RequestID   string
	Host        string
	Path        string
	StatusCode  int
	CacheStatus string
	RemoteIP    string
	CountryCode string
	Referer     string
	UserAgent   string
}

// LogQuery describes one page of request logs to fetch.
type LogQuery struct {
	From       time.Time
	To         time.Time
	Host       string
	PathPrefix string
	Status     string
	Limit      int
	Offset     int
}

// LogPage is one page of request logs.
type LogPage struct {
	Entries    []LogEntry
	NextOffset int
	HasMore    bool
}

// LogSource fetches request logs. Implementations are responsible for the
// provider-specific HTTP protocol.
type LogSource interface {
	List(ctx context.Context, query LogQuery) (LogPage, error)
}

// NormalizeHost lowercases a host and removes a trailing dot.
func NormalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

// RequestKey returns a stable key for deduplicating log rows within one run.
func RequestKey(entry LogEntry) string {
	if entry.RequestID != "" {
		return entry.RequestID
	}

	parts := []string{
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		NormalizeHost(entry.Host),
		entry.Path,
		strconv.Itoa(entry.StatusCode),
		entry.RemoteIP,
		entry.UserAgent,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
