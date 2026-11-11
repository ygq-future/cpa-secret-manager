package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"

	"cpa-secret-manager/internal/pricing"
)

// PriceSchemaVersion is the current reference-price cache schema. The cache is
// derived data: it can be deleted at any time and the next explicit sync
// rebuilds it from the upstream catalog.
const PriceSchemaVersion = 1

// PriceDocument is the compact reference-price cache. It stores only the models
// the page asked about during the last sync, never the upstream catalog.
type PriceDocument struct {
	SchemaVersion int                           `json:"schema_version"`
	Source        string                        `json:"source,omitempty"`
	FetchedAt     string                        `json:"fetched_at,omitempty"`
	ETag          string                        `json:"etag,omitempty"`
	Known         []string                      `json:"known,omitempty"`
	Models        map[string]pricing.Suggestion `json:"models,omitempty"`
}

// PriceStore owns the reference-price cache and its atomic persistence.
type PriceStore struct {
	mu   sync.RWMutex
	path string
	doc  PriceDocument
}

// NewPriceDocument returns an empty price document.
func NewPriceDocument() PriceDocument {
	return PriceDocument{SchemaVersion: PriceSchemaVersion, Models: map[string]pricing.Suggestion{}}
}

// LoadPrices reads the price cache at path. A missing file yields an empty
// document; a malformed file yields an empty document plus the decode error so
// the caller can surface it without losing service.
func LoadPrices(path string) (*PriceStore, error) {
	store := &PriceStore{path: path, doc: NewPriceDocument()}
	if strings.TrimSpace(path) == "" {
		return store, errors.New("state: empty price path")
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("state: read %s: %w", path, err)
	}

	var doc PriceDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return store, fmt.Errorf("state: decode %s: %w", path, err)
	}
	if doc.Models == nil {
		doc.Models = map[string]pricing.Suggestion{}
	}
	doc.SchemaVersion = PriceSchemaVersion
	store.doc = doc
	return store, nil
}

// Meta returns the source, last successful sync instant and upstream ETag.
func (s *PriceStore) Meta() (source string, fetchedAt string, etag string) {
	if s == nil {
		return "", "", ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doc.Source, s.doc.FetchedAt, s.doc.ETag
}

// Suggestions returns a copy of the cached suggestions, keyed by normalized
// model name.
func (s *PriceStore) Suggestions() map[string]pricing.Suggestion {
	if s == nil {
		return map[string]pricing.Suggestion{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSuggestions(s.doc.Models)
}

// Known returns the normalized model names covered by the last successful
// catalog fetch, including models that had no matching offer. A conditional
// refresh can only trust the cache when every requested model is in this set.
func (s *PriceStore) Known() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.doc.Known) == 0 {
		return nil
	}
	return append([]string(nil), s.doc.Known...)
}

// Replace stores a new sync result. The caller owns the maps; this method keeps
// copies so later mutations cannot leak into the cache. Known names are stored
// normalized so a spelling change cannot defeat the conditional-request guard.
func (s *PriceStore) Replace(source string, fetchedAt string, etag string, models map[string]pricing.Suggestion, known []string) {
	if s == nil {
		return
	}
	knownCopy := make([]string, 0, len(known))
	seen := make(map[string]struct{}, len(known))
	for _, name := range known {
		normalized := pricing.NormalizeModel(name)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		knownCopy = append(knownCopy, normalized)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc = PriceDocument{
		SchemaVersion: PriceSchemaVersion,
		Source:        strings.TrimSpace(source),
		FetchedAt:     strings.TrimSpace(fetchedAt),
		ETag:          strings.TrimSpace(etag),
		Known:         knownCopy,
		Models:        cloneSuggestions(models),
	}
}

// Snapshot returns a deep copy of the cache document for persistence or
// inspection.
func (s *PriceStore) Snapshot() PriceDocument {
	if s == nil {
		return NewPriceDocument()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc := s.doc
	doc.Models = cloneSuggestions(s.doc.Models)
	doc.Known = append([]string(nil), s.doc.Known...)
	return doc
}

// SaveAtomic writes the cache through a temporary file and an atomic rename, so
// readers never observe a partially written document.
func (s *PriceStore) SaveAtomic() error {
	if s == nil {
		return errors.New("state: nil price store")
	}
	doc := s.Snapshot()
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("state: encode price document: %w", err)
	}
	return writeAtomic(s.path, append(data, '\n'))
}

func cloneSuggestions(source map[string]pricing.Suggestion) map[string]pricing.Suggestion {
	if len(source) == 0 {
		return map[string]pricing.Suggestion{}
	}
	out := make(map[string]pricing.Suggestion, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
