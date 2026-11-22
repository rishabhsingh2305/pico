package entities

import (
	"errors"
	"time"
)

// ErrAnalyticsSourceNotFound is returned when an analytics source is missing.
var ErrAnalyticsSourceNotFound = errors.New("analytics source not found")

// ErrAnalyticsKeyTaken is returned when a generated pixel key collides.
var ErrAnalyticsKeyTaken = errors.New("analytics key already in use")

// AnalyticsSource is a named public capture pixel.
type AnalyticsSource struct {
	ID          string
	OrgID       string
	Name        string
	Description string
	Key         string
	Events      []string
	DisabledAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Disabled reports whether the source currently accepts captures.
func (s AnalyticsSource) Disabled() bool {
	return s.DisabledAt != nil
}

// AllowsEvent reports whether the source accepts event.
func (s AnalyticsSource) AllowsEvent(event string) bool {
	for _, allowed := range s.Events {
		if allowed == event {
			return true
		}
	}
	return false
}
