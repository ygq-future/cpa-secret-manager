// Package usage attributes host usage records to proxy API keys and keeps the
// cumulative token counters shown per key and per model.
package usage

import (
	"sync"
	"time"
)

// Detail is the token accounting reported by the host for one request.
type Detail struct {
	Input         int64
	Output        int64
	Reasoning     int64
	Cached        int64
	CacheRead     int64
	CacheCreation int64
	Total         int64
}

// Counters is the cumulative token accounting shared by key and model rows.
type Counters struct {
	Requests      int64 `json:"requests"`
	Failed        int64 `json:"failed"`
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
	Reasoning     int64 `json:"reasoning"`
	Cached        int64 `json:"cached"`
	CacheRead     int64 `json:"cache_read"`
	CacheCreation int64 `json:"cache_creation"`
	Total         int64 `json:"total"`
}

// Add folds one request into the counters.
func (c *Counters) Add(detail Detail, failed bool) {
	if c == nil {
		return
	}
	c.Requests++
	if failed {
		c.Failed++
	}
	c.Input += detail.Input
	c.Output += detail.Output
	c.Reasoning += detail.Reasoning
	c.Cached += detail.Cached
	c.CacheRead += detail.CacheRead
	c.CacheCreation += detail.CacheCreation
	c.Total += detail.Total
}

// IsZero reports whether nothing was ever accounted here.
func (c Counters) IsZero() bool {
	return c == Counters{}
}

// ModelUsage is the per-model breakdown for one key.
type ModelUsage struct {
	Counters
	FirstSeen time.Time `json:"first_seen,omitempty"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// KeyUsage is the aggregate for one key index.
type KeyUsage struct {
	Counters
	FirstSeen time.Time             `json:"first_seen,omitempty"`
	LastSeen  time.Time             `json:"last_seen,omitempty"`
	Models    map[string]ModelUsage `json:"models,omitempty"`
}

// Record is one normalized usage observation.
type Record struct {
	// Hash is the SHA-256 index of the client proxy API key.
	Hash string
	// Model classifies the request; an empty model is grouped as UnknownModel.
	Model  string
	Failed bool
	Detail Detail
	At     time.Time
}

// UnknownModel groups records that carried no model name.
const UnknownModel = "unknown"

// Aggregator accumulates usage records. It is safe for concurrent use and owns
// the live counters, which are materialized into the state document by the
// application layer.
type Aggregator struct {
	mu           sync.Mutex
	keys         map[string]*KeyUsage
	unattributed int64
	dirty        bool
}

// NewAggregator returns an empty aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{keys: make(map[string]*KeyUsage)}
}

// Observe folds one record into the aggregate.
func (a *Aggregator) Observe(record Record) {
	if a == nil || record.Hash == "" {
		return
	}
	model := record.Model
	if model == "" {
		model = UnknownModel
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.keys[record.Hash]
	if !ok {
		entry = &KeyUsage{Models: make(map[string]ModelUsage)}
		a.keys[record.Hash] = entry
	}
	entry.Add(record.Detail, record.Failed)
	// FirstSeen only moves backward to a genuinely earlier observation, so a
	// replayed or out-of-order record cannot corrupt the window.
	if entry.FirstSeen.IsZero() || (!record.At.IsZero() && record.At.Before(entry.FirstSeen)) {
		entry.FirstSeen = record.At
	}
	if record.At.After(entry.LastSeen) {
		entry.LastSeen = record.At
	}

	modelUsage := entry.Models[model]
	modelUsage.Add(record.Detail, record.Failed)
	if modelUsage.FirstSeen.IsZero() || (!record.At.IsZero() && record.At.Before(modelUsage.FirstSeen)) {
		modelUsage.FirstSeen = record.At
	}
	if record.At.After(modelUsage.LastSeen) {
		modelUsage.LastSeen = record.At
	}
	entry.Models[model] = modelUsage

	a.dirty = true
}

// ObserveUnattributed counts a record that carried no attributable key.
func (a *Aggregator) ObserveUnattributed() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.unattributed++
	a.dirty = true
}

// Seed replaces the aggregate with previously persisted counters.
func (a *Aggregator) Seed(keys map[string]KeyUsage, unattributed int64) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	a.keys = make(map[string]*KeyUsage, len(keys))
	for hash, entry := range keys {
		copied := entry
		copied.Models = make(map[string]ModelUsage, len(entry.Models))
		for model, modelUsage := range entry.Models {
			copied.Models[model] = modelUsage
		}
		a.keys[hash] = &copied
	}
	a.unattributed = unattributed
	a.dirty = false
}

// Snapshot returns a deep copy of the aggregate.
func (a *Aggregator) Snapshot() (map[string]KeyUsage, int64) {
	if a == nil {
		return map[string]KeyUsage{}, 0
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]KeyUsage, len(a.keys))
	for hash, entry := range a.keys {
		copied := *entry
		copied.Models = make(map[string]ModelUsage, len(entry.Models))
		for model, modelUsage := range entry.Models {
			copied.Models[model] = modelUsage
		}
		out[hash] = copied
	}
	return out, a.unattributed
}

// Dirty reports whether the aggregate changed since the last MarkClean.
func (a *Aggregator) Dirty() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dirty
}

// MarkClean clears the dirty flag after a successful persist.
func (a *Aggregator) MarkClean() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.dirty = false
}
