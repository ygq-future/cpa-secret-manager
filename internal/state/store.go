package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cpa-secret-manager/internal/remarks"
)

// SchemaVersion is the current plugin state document schema.
const SchemaVersion = 1

// AppConfig carries plugin-owned business settings that must not live in the
// host config.yaml, see ADR-0001.
type AppConfig struct {
	// UsageEnabled controls whether usage records are aggregated.
	UsageEnabled bool `json:"usage_enabled"`
}

// DefaultAppConfig returns the canonical business defaults.
func DefaultAppConfig() AppConfig {
	return AppConfig{UsageEnabled: true}
}

// Metrics holds plugin-level counters that are not attributed to any key.
type Metrics struct {
	// UnattributedRequests counts usage records without a matching managed key.
	UnattributedRequests int64 `json:"unattributed_requests"`
}

// Document is the persisted plugin state.
type Document struct {
	SchemaVersion int                      `json:"schema_version"`
	Remarks       map[string]remarks.Entry `json:"remarks,omitempty"`
	AppConfig     AppConfig                `json:"app_config"`
	Metrics       Metrics                  `json:"metrics"`
}

// Store owns the plugin state document and its atomic persistence.
type Store struct {
	mu   sync.RWMutex
	path string
	doc  Document
}

// Load reads the state document at path. A missing file yields a default
// document; a malformed file yields a default document plus the decode error so
// callers can surface it without losing service.
func Load(path string) (*Store, error) {
	store := &Store{path: path, doc: NewDocument()}
	if strings.TrimSpace(path) == "" {
		return store, errors.New("state: empty state path")
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("state: read %s: %w", path, err)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return store, fmt.Errorf("state: decode %s: %w", path, err)
	}
	store.doc = applyDefaults(doc, data)
	return store, nil
}

// NewDocument returns an empty document with canonical defaults.
func NewDocument() Document {
	return Document{SchemaVersion: SchemaVersion, AppConfig: DefaultAppConfig()}
}

// Path returns the configured state file path.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// AppConfig returns a copy of the current business settings.
func (s *Store) AppConfig() AppConfig {
	if s == nil {
		return DefaultAppConfig()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doc.AppConfig
}

// SetAppConfig replaces the business settings.
func (s *Store) SetAppConfig(cfg AppConfig) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.AppConfig = cfg
}

// Metrics returns the current plugin-level counters.
func (s *Store) Metrics() Metrics {
	if s == nil {
		return Metrics{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doc.Metrics
}

// AddUnattributed increments the unattributed request counter.
func (s *Store) AddUnattributed(delta int64) {
	if s == nil || delta == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Metrics.UnattributedRequests += delta
}

// Remarks returns a copy of the remark index.
func (s *Store) Remarks() map[string]remarks.Entry {
	if s == nil {
		return map[string]remarks.Entry{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]remarks.Entry, len(s.doc.Remarks))
	for hash, entry := range s.doc.Remarks {
		out[hash] = entry
	}
	return out
}

// Remark returns the remark stored for one hash index.
func (s *Store) Remark(hash string) (remarks.Entry, bool) {
	if s == nil {
		return remarks.Entry{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.doc.Remarks[hash]
	return entry, ok
}

// SetRemark stores or replaces one remark.
func (s *Store) SetRemark(hash string, entry remarks.Entry) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc.Remarks == nil {
		s.doc.Remarks = make(map[string]remarks.Entry)
	}
	s.doc.Remarks[hash] = entry
}

// DeleteRemark removes one remark, reporting whether it existed.
func (s *Store) DeleteRemark(hash string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.doc.Remarks[hash]; !ok {
		return false
	}
	delete(s.doc.Remarks, hash)
	return true
}

// Snapshot returns a deep copy of the document for persistence or inspection.
func (s *Store) Snapshot() Document {
	if s == nil {
		return NewDocument()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doc.clone()
}

// SaveAtomic writes the document to disk through a temporary file and an
// atomic rename, so readers never observe a partially written document.
func (s *Store) SaveAtomic() error {
	if s == nil {
		return errors.New("state: nil store")
	}
	doc := s.Snapshot()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("state: encode document: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("state: create directory %s: %w", dir, err)
		}
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("state: write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("state: replace %s: %w", s.path, err)
	}
	return nil
}

// applyDefaults fills canonical defaults for sections that are absent from the
// persisted document. Presence is probed on the raw JSON because a value type
// cannot distinguish "false" from "not written".
func applyDefaults(doc Document, raw []byte) Document {
	doc.SchemaVersion = SchemaVersion

	var sections map[string]json.RawMessage
	if err := json.Unmarshal(raw, &sections); err != nil {
		return doc
	}
	rawAppConfig, present := sections["app_config"]
	if !present {
		doc.AppConfig = DefaultAppConfig()
		return doc
	}
	var appConfig map[string]json.RawMessage
	if err := json.Unmarshal(rawAppConfig, &appConfig); err != nil {
		return doc
	}
	if _, ok := appConfig["usage_enabled"]; !ok {
		doc.AppConfig = DefaultAppConfig()
	}
	return doc
}

func (doc Document) clone() Document {
	if len(doc.Remarks) > 0 {
		remarksCopy := make(map[string]remarks.Entry, len(doc.Remarks))
		for hash, entry := range doc.Remarks {
			remarksCopy[hash] = entry
		}
		doc.Remarks = remarksCopy
	}
	return doc
}
