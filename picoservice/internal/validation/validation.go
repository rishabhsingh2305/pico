// Package validation provides shared input validation errors and pagination
// rules for domain services.
package validation

import "strings"

const (
	// DefaultLimit is the default page size for list endpoints.
	DefaultLimit = 20
	// MaxLimit is the largest accepted page size.
	MaxLimit = 100

	nameMaxLength = 120
)

// Error describes invalid input. Field identifies the offending input and
// Message is safe to return to API clients.
type Error struct {
	Field   string
	Message string
}

// Error implements error.
func (e *Error) Error() string {
	return e.Message
}

// Name trims and validates a display name shared by users, orgs, and projects.
func Name(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", &Error{Field: "name", Message: "name is required"}
	}
	if len(name) > nameMaxLength {
		return "", &Error{Field: "name", Message: "name must be at most 120 characters"}
	}
	return name, nil
}

// Currency normalizes and validates a three-letter ISO currency code.
func Currency(raw string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(raw))
	if len(currency) != 3 {
		return "", &Error{Field: "currency", Message: "currency must be a 3-letter code"}
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return "", &Error{Field: "currency", Message: "currency must be a 3-letter code"}
		}
	}
	return currency, nil
}

// Pagination normalizes limit and offset for list endpoints.
func Pagination(limit, offset int) (int, int, error) {
	switch {
	case limit < 0:
		return 0, 0, &Error{Field: "limit", Message: "limit must be a positive integer"}
	case limit == 0:
		limit = DefaultLimit
	case limit > MaxLimit:
		return 0, 0, &Error{Field: "limit", Message: "limit must be at most 100"}
	}

	if offset < 0 {
		return 0, 0, &Error{Field: "offset", Message: "offset must be non-negative"}
	}
	return limit, offset, nil
}
