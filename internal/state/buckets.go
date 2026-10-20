package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"

	"cpa-secret-manager/internal/usage"
)

// BucketSchemaVersion is the current bucket document schema.
const BucketSchemaVersion = 1

// BucketDocument is the persisted time-bucketed usage accounting. It lives in
// its own file beside the state document: the settings, remarks and quota rules
// stay small and readable, while the history - the part that grows with traffic
// - is written on its own.
type BucketDocument struct {
	SchemaVersion int                                `json:"schema_version"`
	Buckets       map[string]map[string]usage.Bucket `json:"buckets,omitempty"`
}

// BucketStore owns the bucket document and its atomic persistence.
type BucketStore struct {
	mu   sync.RWMutex
	path string
	doc  BucketDocument
}

// NewBucketDocument returns an empty bucket document.
func NewBucketDocument() BucketDocument {
	return BucketDocument{SchemaVersion: BucketSchemaVersion, Buckets: map[string]map[string]usage.Bucket{}}
}

// LoadBuckets reads the bucket document at path. A missing file yields an empty
// document; a malformed file yields an empty document plus the decode error so
// the caller can surface it without losing service.
func LoadBuckets(path string) (*BucketStore, error) {
	store := &BucketStore{path: path, doc: NewBucketDocument()}
	if strings.TrimSpace(path) == "" {
		return store, errors.New("state: empty bucket path")
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("state: read %s: %w", path, err)
	}

	var doc BucketDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return store, fmt.Errorf("state: decode %s: %w", path, err)
	}
	if doc.Buckets == nil {
		doc.Buckets = map[string]map[string]usage.Bucket{}
	}
	doc.SchemaVersion = BucketSchemaVersion
	store.doc = doc
	return store, nil
}

// Path returns the storage path.
func (s *BucketStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Buckets returns a deep copy of the persisted buckets.
func (s *BucketStore) Buckets() map[string]map[string]usage.Bucket {
	if s == nil {
		return map[string]map[string]usage.Bucket{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneBuckets(s.doc.Buckets)
}

// ReplaceBuckets materializes the live aggregate into the document. The caller
// owns the buckets; this method only stores a copy.
func (s *BucketStore) ReplaceBuckets(buckets map[string]map[string]usage.Bucket) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Buckets = cloneBuckets(buckets)
}

// Backup writes the current bucket document to <path>.bak so a destructive
// change - a key forgotten on the page's request - stays recoverable by hand.
// It mirrors Store.Backup: a recovery copy of the moment before the deletion,
// not a history archive.
func (s *BucketStore) Backup() error {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return errors.New("state: nil bucket store")
	}
	s.mu.RLock()
	data, err := json.MarshalIndent(s.doc, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("state: encode bucket document: %w", err)
	}
	return writeAtomic(s.path+".bak", append(data, '\n'))
}

// SaveAtomic writes the bucket document through a temporary file and an atomic
// rename, so readers never observe a partially written document.
func (s *BucketStore) SaveAtomic() error {
	if s == nil {
		return errors.New("state: nil bucket store")
	}
	s.mu.RLock()
	data, err := json.MarshalIndent(s.doc, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("state: encode bucket document: %w", err)
	}
	return writeAtomic(s.path, append(data, '\n'))
}

func cloneBuckets(source map[string]map[string]usage.Bucket) map[string]map[string]usage.Bucket {
	if len(source) == 0 {
		return map[string]map[string]usage.Bucket{}
	}
	out := make(map[string]map[string]usage.Bucket, len(source))
	for hash, byPeriod := range source {
		copied := make(map[string]usage.Bucket, len(byPeriod))
		for period, bucket := range byPeriod {
			entry := usage.Bucket{Counters: bucket.Counters, Models: make(map[string]usage.Counters, len(bucket.Models))}
			for model, counters := range bucket.Models {
				entry.Models[model] = counters
			}
			copied[period] = entry
		}
		out[hash] = copied
	}
	return out
}
