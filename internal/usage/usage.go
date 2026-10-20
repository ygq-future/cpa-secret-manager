// Package usage attributes host usage records to proxy API keys and keeps the
// hourly token buckets that every per-key and per-model view is computed from.
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

// Counters is the token accounting shared by key and model rows.
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

// Merge folds another counter set into c.
func (c *Counters) Merge(other Counters) {
	if c == nil {
		return
	}
	c.Requests += other.Requests
	c.Failed += other.Failed
	c.Input += other.Input
	c.Output += other.Output
	c.Reasoning += other.Reasoning
	c.Cached += other.Cached
	c.CacheRead += other.CacheRead
	c.CacheCreation += other.CacheCreation
	c.Total += other.Total
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

// KeyUsage is the aggregate for one key over a queried window.
type KeyUsage struct {
	Counters
	FirstSeen time.Time             `json:"first_seen,omitempty"`
	LastSeen  time.Time             `json:"last_seen,omitempty"`
	Models    map[string]ModelUsage `json:"models,omitempty"`
}

// Bucket is the usage one key observed inside one hourly slice, split by model.
// Buckets are the only time-attributed accounting the plugin keeps: a range
// query merges the buckets that overlap the window instead of scanning
// individual requests.
type Bucket struct {
	Counters
	Models map[string]Counters `json:"models,omitempty"`
	// LastAt is the exact instant of the latest request folded into this bucket.
	// Buckets written before this field existed leave it zero; readers fall back
	// to the bucket start, a lower bound that is never in the future.
	LastAt time.Time `json:"last_at,omitzero"`
}

// PeriodLayout identifies a bucket by its UTC hour.
const PeriodLayout = "2006-01-02T15"

// BucketSpan is the duration one bucket covers.
const BucketSpan = time.Hour

// PeriodKey returns the key of the bucket an instant belongs to.
func PeriodKey(at time.Time) string {
	return at.UTC().Truncate(BucketSpan).Format(PeriodLayout)
}

// ParsePeriod parses a bucket key back into the UTC instant that starts it.
func ParsePeriod(key string) (time.Time, bool) {
	at, err := time.ParseInLocation(PeriodLayout, key, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

// Record is one normalized usage observation.
type Record struct {
	// Hash is the SHA-256 index of the client proxy API key.
	Hash string
	// Model classifies the request; an empty model is grouped as UnknownModel.
	Model          string
	Failed         bool
	FailureStatus  int
	FailureMessage string
	Detail         Detail
	At             time.Time
}

// UnknownModel groups records that carried no model name.
const UnknownModel = "unknown"

// CloneBucket returns a deep copy of one bucket. Every field, including the
// precise last-observation instant, must survive persistence and snapshots.
func CloneBucket(bucket Bucket) Bucket {
	copied := Bucket{
		Counters: bucket.Counters,
		LastAt:   bucket.LastAt,
		Models:   make(map[string]Counters, len(bucket.Models)),
	}
	for model, counters := range bucket.Models {
		copied.Models[model] = counters
	}
	return copied
}

// Aggregator accumulates usage records into hourly buckets. It is safe for
// concurrent use and owns the live accounting, which the application layer
// materializes into the bucket document.
type Aggregator struct {
	mu      sync.Mutex
	buckets map[string]map[string]*Bucket
	dirty   bool
}

// NewAggregator returns an empty aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{buckets: make(map[string]map[string]*Bucket)}
}

// Observe folds one record into the bucket of its hour.
func (a *Aggregator) Observe(record Record) {
	if a == nil || record.Hash == "" {
		return
	}
	model := record.Model
	if model == "" {
		model = UnknownModel
	}
	at := record.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	period := PeriodKey(at)

	a.mu.Lock()
	defer a.mu.Unlock()

	byPeriod, ok := a.buckets[record.Hash]
	if !ok {
		byPeriod = make(map[string]*Bucket)
		a.buckets[record.Hash] = byPeriod
	}
	bucket, ok := byPeriod[period]
	if !ok {
		bucket = &Bucket{Models: make(map[string]Counters)}
		byPeriod[period] = bucket
	}
	bucket.Add(record.Detail, record.Failed)
	modelCounters := bucket.Models[model]
	modelCounters.Add(record.Detail, record.Failed)
	bucket.Models[model] = modelCounters
	if bucket.LastAt.Before(at) {
		bucket.LastAt = at
	}

	a.dirty = true
}

// Range merges every bucket that overlaps [start, end) into per-key usage. A
// zero bound means "unbounded". Buckets are hourly, so a window that starts or
// ends mid-hour includes that whole hour.
func (a *Aggregator) Range(start, end time.Time) map[string]KeyUsage {
	if a == nil {
		return map[string]KeyUsage{}
	}
	if !start.IsZero() && !end.IsZero() && !end.After(start) {
		return map[string]KeyUsage{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]KeyUsage, len(a.buckets))
	for hash, byPeriod := range a.buckets {
		merged := KeyUsage{Models: make(map[string]ModelUsage)}
		matched := false
		for period, bucket := range byPeriod {
			at, ok := ParsePeriod(period)
			if !ok {
				continue
			}
			if !overlaps(at, start, end) {
				continue
			}
			matched = true
			merged.Merge(bucket.Counters)
			for model, counters := range bucket.Models {
				modelUsage := merged.Models[model]
				modelUsage.Merge(counters)
				merged.Models[model] = modelUsage
			}
			if merged.FirstSeen.IsZero() || at.Before(merged.FirstSeen) {
				merged.FirstSeen = at
			}
			observed := bucket.LastAt
			if observed.IsZero() {
				observed = at
			}
			if observed.After(merged.LastSeen) {
				merged.LastSeen = observed
			}
		}
		if matched {
			out[hash] = merged
		}
	}
	return out
}

// overlaps reports whether the bucket starting at at shares any instant with
// [start, end). Zero bounds are unbounded.
func overlaps(at, start, end time.Time) bool {
	finish := at.Add(BucketSpan)
	if !start.IsZero() && !finish.After(start) {
		return false
	}
	if !end.IsZero() && !at.Before(end) {
		return false
	}
	return true
}

// LastSeen reports the latest observed request instant per key across every
// retained bucket. It ignores query windows on purpose: "last call" answers a
// lifetime question, so a key that was active before the selected range still
// has an answer.
func (a *Aggregator) LastSeen() map[string]time.Time {
	if a == nil {
		return map[string]time.Time{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]time.Time, len(a.buckets))
	for hash, byPeriod := range a.buckets {
		var latest time.Time
		for period, bucket := range byPeriod {
			at, ok := ParsePeriod(period)
			if !ok {
				continue
			}
			observed := bucket.LastAt
			if observed.IsZero() {
				observed = at
			}
			if observed.After(latest) {
				latest = observed
			}
		}
		if !latest.IsZero() {
			out[hash] = latest
		}
	}
	return out
}

// Prune drops every bucket whose hour ended at or before cutoff and reports how
// many buckets it removed.
func (a *Aggregator) Prune(cutoff time.Time) int {
	if a == nil || cutoff.IsZero() {
		return 0
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	removed := 0
	for hash, byPeriod := range a.buckets {
		for period := range byPeriod {
			at, ok := ParsePeriod(period)
			if !ok || !at.Add(BucketSpan).After(cutoff) {
				delete(byPeriod, period)
				removed++
			}
		}
		if len(byPeriod) == 0 {
			delete(a.buckets, hash)
		}
	}
	if removed > 0 {
		a.dirty = true
	}
	return removed
}

// SeedBuckets replaces the aggregate with previously persisted buckets.
func (a *Aggregator) SeedBuckets(buckets map[string]map[string]Bucket) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	a.buckets = make(map[string]map[string]*Bucket, len(buckets))
	for hash, byPeriod := range buckets {
		copied := make(map[string]*Bucket, len(byPeriod))
		for period, bucket := range byPeriod {
			entry := CloneBucket(bucket)
			copied[period] = &entry
		}
		a.buckets[hash] = copied
	}
	a.dirty = false
}

// SnapshotBuckets returns a deep copy of the buckets for persistence.
func (a *Aggregator) SnapshotBuckets() map[string]map[string]Bucket {
	if a == nil {
		return map[string]map[string]Bucket{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]map[string]Bucket, len(a.buckets))
	for hash, byPeriod := range a.buckets {
		copied := make(map[string]Bucket, len(byPeriod))
		for period, bucket := range byPeriod {
			copied[period] = CloneBucket(*bucket)
		}
		out[hash] = copied
	}
	return out
}

// Hashes returns the keys that currently own at least one bucket.
func (a *Aggregator) Hashes() map[string]struct{} {
	if a == nil {
		return map[string]struct{}{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]struct{}, len(a.buckets))
	for hash := range a.buckets {
		out[hash] = struct{}{}
	}
	return out
}

// Models returns every model name seen in any bucket.
func (a *Aggregator) Models() map[string]struct{} {
	if a == nil {
		return map[string]struct{}{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[string]struct{})
	for _, byPeriod := range a.buckets {
		for _, bucket := range byPeriod {
			for model := range bucket.Models {
				if model != "" {
					out[model] = struct{}{}
				}
			}
		}
	}
	return out
}

// Remove drops every bucket of one key, reporting whether it had any. The
// application layer calls it when a key disappears for good.
func (a *Aggregator) Remove(hash string) bool {
	if a == nil || hash == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.buckets[hash]; !ok {
		return false
	}
	delete(a.buckets, hash)
	a.dirty = true
	return true
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
