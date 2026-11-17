package errorslog

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrorEntry represents a single failed request event.
type ErrorEntry struct {
	Time       time.Time `json:"time"`
	Model      string    `json:"model"`
	AuthID     string    `json:"auth_id,omitempty"`
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
}

// documentSchemaVersion is the persisted layout version. Version 2 stores
// failure messages as plain text; version 1 documents kept the HTML-escaped
// upstream body the host reports (&#34; in place of a quote) and are decoded once
// when they are loaded.
const documentSchemaVersion = 2

// Document is the persisted format of the errors log.
type Document struct {
	SchemaVersion int                     `json:"schema_version"`
	Failures      map[string][]ErrorEntry `json:"failures,omitempty"`
}

// Store owns the error log entries and their atomic persistence.
type Store struct {
	mu      sync.RWMutex
	path    string
	entries map[string][]ErrorEntry
	dirty   bool
}

// Load loads the error log from the given path.
// If the file does not exist, a new empty store is returned.
func Load(path string) (*Store, error) {
	store := &Store{
		path:    path,
		entries: make(map[string][]ErrorEntry),
	}
	if strings.TrimSpace(path) == "" {
		return store, errors.New("errorslog: empty path")
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("errorslog: read %s: %w", path, err)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return store, fmt.Errorf("errorslog: decode %s: %w", path, err)
	}

	if doc.Failures != nil {
		store.entries = doc.Failures
	}
	if doc.SchemaVersion < documentSchemaVersion {
		store.decodeMessages()
	}
	return store, nil
}

// decodeMessages repairs documents written before the schema stored plain text:
// each message is HTML-unescaped exactly once, and the store is marked dirty so
// the next save persists the readable text under the current schema version.
func (s *Store) decodeMessages() {
	for _, list := range s.entries {
		for i := range list {
			list[i].Message = html.UnescapeString(list[i].Message)
		}
	}
	s.dirty = true
}

// Path returns the storage path.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Record appends a failure entry for the given key hash.
func (s *Store) Record(hash string, entry ErrorEntry) {
	if s == nil || strings.TrimSpace(hash) == "" {
		return
	}
	if entry.Time.IsZero() {
		entry.Time = time.Now().UTC()
	} else {
		entry.Time = entry.Time.UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[hash] = append(s.entries[hash], entry)
	s.dirty = true
}

// Query returns error entries for a specific key hash within [start, end).
// If start is zero, no lower bound is checked. If end is zero, no upper bound is checked.
// Results are returned in descending chronological order (newest first).
func (s *Store) Query(hash string, start, end time.Time) []ErrorEntry {
	if s == nil || strings.TrimSpace(hash) == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, ok := s.entries[hash]
	if !ok || len(list) == 0 {
		return nil
	}

	var res []ErrorEntry
	for _, item := range list {
		if !start.IsZero() && item.Time.Before(start) {
			continue
		}
		if !end.IsZero() && !item.Time.Before(end) {
			continue
		}
		res = append(res, item)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Time.After(res[j].Time)
	})
	return res
}

// QueryAll returns error entries grouped by key hash within [start, end).
// Results in each group are sorted newest first.
func (s *Store) QueryAll(start, end time.Time) map[string][]ErrorEntry {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string][]ErrorEntry, len(s.entries))
	for hash, list := range s.entries {
		var matched []ErrorEntry
		for _, item := range list {
			if !start.IsZero() && item.Time.Before(start) {
				continue
			}
			if !end.IsZero() && !item.Time.Before(end) {
				continue
			}
			matched = append(matched, item)
		}
		if len(matched) > 0 {
			sort.Slice(matched, func(i, j int) bool {
				return matched[i].Time.After(matched[j].Time)
			})
			out[hash] = matched
		}
	}
	return out
}

// Prune removes error entries older than cutoff.
// Returns the number of removed entries.
func (s *Store) Prune(cutoff time.Time) int {
	if s == nil || cutoff.IsZero() {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for hash, list := range s.entries {
		var kept []ErrorEntry
		for _, item := range list {
			if item.Time.Before(cutoff) {
				removed++
			} else {
				kept = append(kept, item)
			}
		}
		if len(kept) == 0 {
			delete(s.entries, hash)
		} else if len(kept) < len(list) {
			s.entries[hash] = kept
		}
	}

	if removed > 0 {
		s.dirty = true
	}
	return removed
}

// ClearFilter constrains which error entries are removed by ClearBefore.
// Before is required; Hash, AuthID, Model, and StatusCode are optional filters.
type ClearFilter struct {
	Before     time.Time
	Hash       string
	AuthID     string
	Model      string
	StatusCode *int
}
// ClearedEntry identifies a removed error log entry for downstream counter sync.
type ClearedEntry struct {
	Hash  string
	Time  time.Time
	Model string
}

// ClearBefore removes error entries recorded before filter.Before that match all
// supplied filter fields, returning the number of removed entries.
func (s *Store) ClearBefore(filter ClearFilter) int {
	return len(s.ClearBeforeEntries(filter))
}

// ClearBeforeEntries removes error entries recorded before filter.Before that
// match all supplied filter fields and returns the removed entries.
func (s *Store) ClearBeforeEntries(filter ClearFilter) []ClearedEntry {
	if s == nil || filter.Before.IsZero() {
		return nil
	}
	before := filter.Before.UTC()
	targetHash := strings.TrimSpace(filter.Hash)

	s.mu.Lock()
	defer s.mu.Unlock()

	var removed []ClearedEntry
	clearHash := func(hash string, list []ErrorEntry) {
		var kept []ErrorEntry
		for _, item := range list {
			if matchesClearFilter(item, before, filter.AuthID, filter.Model, filter.StatusCode) {
				removed = append(removed, ClearedEntry{
					Hash:  hash,
					Time:  item.Time,
					Model: item.Model,
				})
			} else {
				kept = append(kept, item)
			}
		}
		if len(kept) == 0 {
			delete(s.entries, hash)
		} else if len(kept) < len(list) {
			s.entries[hash] = kept
		}
	}

	if targetHash != "" {
		if list, ok := s.entries[targetHash]; ok {
			clearHash(targetHash, list)
		}
	} else {
		for hash, list := range s.entries {
			clearHash(hash, list)
		}
	}

	if len(removed) > 0 {
		s.dirty = true
	}
	return removed
}

func matchesClearFilter(item ErrorEntry, before time.Time, authID, model string, statusCode *int) bool {
	if !item.Time.Before(before) {
		return false
	}
	if authID != "" && item.AuthID != authID {
		return false
	}
	if model != "" && item.Model != model {
		return false
	}
	if statusCode != nil && item.StatusCode != *statusCode {
		return false
	}
	return true
}

// Forget removes all error entries for the given key hash.
func (s *Store) Forget(hash string) bool {
	if s == nil || strings.TrimSpace(hash) == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[hash]; !ok {
		return false
	}
	delete(s.entries, hash)
	s.dirty = true
	return true
}

// Dirty reports whether there are unsaved changes.
func (s *Store) Dirty() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

// Save writes the error log to disk atomically.
func (s *Store) Save() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(s.path) == "" {
		return errors.New("errorslog: empty path")
	}

	doc := Document{
		SchemaVersion: documentSchemaVersion,
		Failures:      s.entries,
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("errorslog: encode: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("errorslog: mkdir %s: %w", dir, err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("errorslog: write %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("errorslog: commit %s: %w", s.path, err)
	}

	s.dirty = false
	return nil
}
