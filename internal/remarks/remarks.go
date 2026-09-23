// Package remarks owns the remark attached to a proxy API key. Remarks are
// indexed by a SHA-256 digest so the plugin never persists key material.
package remarks

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxLength is the maximum accepted remark length in runes.
const MaxLength = 200

// ErrEmptyKey indicates that no proxy API key was supplied.
var ErrEmptyKey = errors.New("remarks: api key is required")

// Entry is one persisted remark.
type Entry struct {
	Remark    string    `json:"remark"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Index returns the SHA-256 hex digest that keys plugin metadata for a proxy
// API key. The key itself is never persisted.
func Index(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

// Normalize trims surrounding whitespace from remark text.
func Normalize(text string) string {
	return strings.TrimSpace(text)
}

// Validate reports whether normalized remark text is acceptable. An empty
// remark is valid and means "remove the remark".
func Validate(text string) error {
	if utf8.RuneCountInString(text) > MaxLength {
		return fmt.Errorf("remarks: remark must be at most %d characters", MaxLength)
	}
	return nil
}

// ValidateKey reports whether a usable proxy API key was supplied.
func ValidateKey(apiKey string) error {
	if strings.TrimSpace(apiKey) == "" {
		return ErrEmptyKey
	}
	return nil
}
