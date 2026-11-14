// Package expiry owns the expiration instant attached to a proxy API key. The
// instant is stored per SHA-256 hash index; requests from an expired key are
// terminated before they reach an upstream, and the plugin page removes the key
// from the host list on its next refresh.
package expiry

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrEmpty indicates that the submitted expiration value is empty.
var ErrEmpty = errors.New("expiry: expires_at is required")

// Parse reads an RFC3339 instant from a request payload and normalizes it to
// UTC. The page sends the instant of the operator's local wall clock converted
// to UTC, so a persisted value is unambiguous regardless of the host time zone.
func Parse(raw string) (time.Time, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return time.Time{}, ErrEmpty
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}, fmt.Errorf("expiry: expires_at must be an RFC3339 instant: %w", err)
	}
	return parsed.UTC(), nil
}

// Expired reports whether now has reached the instant. A zero instant never
// expires, so a missing entry can be passed through safely.
func Expired(at time.Time, now time.Time) bool {
	if at.IsZero() {
		return false
	}
	return !now.Before(at)
}

// Validate rejects an instant that is not in the future. The check runs on the
// receiving side so a skewed browser clock cannot arm an expiry that already
// passed and trigger an immediate deletion.
func Validate(at time.Time, now time.Time) error {
	if !at.After(now) {
		return errors.New("expiry: expires_at must be in the future")
	}
	return nil
}
