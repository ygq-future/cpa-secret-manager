package expiry

import (
	"errors"
	"testing"
	"time"
)

func TestParseNormalizesToUTC(t *testing.T) {
	parsed, err := Parse("2026-12-01T20:30:00+08:00")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if want := time.Date(2026, 12, 1, 12, 30, 0, 0, time.UTC); !parsed.Equal(want) {
		t.Fatalf("parsed = %s, want %s", parsed, want)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("location = %s, want UTC", parsed.Location())
	}
}

func TestParseRejectsInvalidAndEmpty(t *testing.T) {
	if _, err := Parse("   "); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty Parse error = %v, want ErrEmpty", err)
	}
	if _, err := Parse("2026-12-01 20:30"); err == nil {
		t.Fatal("Parse accepted a non-RFC3339 instant")
	}
}

func TestExpiredUsesInclusiveBoundary(t *testing.T) {
	at := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	if Expired(at, at.Add(-time.Nanosecond)) {
		t.Fatal("instant should not be expired one nanosecond early")
	}
	if !Expired(at, at) {
		t.Fatal("instant should be expired exactly at the boundary")
	}
	if Expired(time.Time{}, at) {
		t.Fatal("a zero instant must never expire")
	}
}

func TestValidateRequiresFutureInstant(t *testing.T) {
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	if err := Validate(now, now); err == nil {
		t.Fatal("Validate accepted the current instant")
	}
	if err := Validate(now.Add(-time.Minute), now); err == nil {
		t.Fatal("Validate accepted a past instant")
	}
	if err := Validate(now.Add(time.Minute), now); err != nil {
		t.Fatalf("Validate rejected a future instant: %v", err)
	}
}
