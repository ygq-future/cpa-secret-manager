package usage

import (
	"sync"
	"testing"
	"time"
)

func TestObserve_AccumulatesKeyAndModelCounters(t *testing.T) {
	aggregator := NewAggregator()
	base := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

	aggregator.Observe(Record{Hash: "h1", Model: "gpt-5.6", At: base,
		Detail: Detail{Input: 10, Output: 20, Reasoning: 3, Cached: 4, Total: 33}})
	aggregator.Observe(Record{Hash: "h1", Model: "gpt-5.6", Failed: true, At: base.Add(time.Minute),
		Detail: Detail{Input: 1, Output: 2, Total: 3}})
	aggregator.Observe(Record{Hash: "h1", Model: "claude-sonnet", At: base.Add(2 * time.Minute),
		Detail: Detail{Input: 5, CacheRead: 7, CacheCreation: 9, Total: 21}})

	snapshot := aggregator.Snapshot()

	entry, ok := snapshot["h1"]
	if !ok {
		t.Fatal("aggregate is missing h1")
	}
	want := Counters{Requests: 3, Failed: 1, Input: 16, Output: 22, Reasoning: 3, Cached: 4, CacheRead: 7, CacheCreation: 9, Total: 57}
	if entry.Counters != want {
		t.Fatalf("key counters = %+v, want %+v", entry.Counters, want)
	}
	if len(entry.Models) != 2 {
		t.Fatalf("models = %d entries, want 2", len(entry.Models))
	}
	if got := entry.Models["gpt-5.6"]; got.Requests != 2 || got.Failed != 1 || got.Total != 36 {
		t.Fatalf("gpt-5.6 counters = %+v, want 2 requests (1 failed) and 36 total tokens", got)
	}
	if !entry.FirstSeen.Equal(base) || !entry.LastSeen.Equal(base.Add(2*time.Minute)) {
		t.Fatalf("window = %v..%v, want %v..%v", entry.FirstSeen, entry.LastSeen, base, base.Add(2*time.Minute))
	}
}

func TestObserve_GroupsMissingModelAsUnknown(t *testing.T) {
	aggregator := NewAggregator()
	aggregator.Observe(Record{Hash: "h1", Detail: Detail{Total: 1}})

	snapshot := aggregator.Snapshot()
	if _, ok := snapshot["h1"].Models[UnknownModel]; !ok {
		t.Fatalf("models = %+v, want an %s bucket", snapshot["h1"].Models, UnknownModel)
	}
}

func TestObserve_KeepsEarliestTimestampOnReplay(t *testing.T) {
	aggregator := NewAggregator()
	later := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	earlier := later.Add(-time.Hour)

	aggregator.Observe(Record{Hash: "h1", Model: "m", At: later})
	aggregator.Observe(Record{Hash: "h1", Model: "m", At: earlier})

	snapshot := aggregator.Snapshot()
	if !snapshot["h1"].FirstSeen.Equal(earlier) {
		t.Fatalf("first_seen = %v, want the earlier observation %v", snapshot["h1"].FirstSeen, earlier)
	}
	if !snapshot["h1"].LastSeen.Equal(later) {
		t.Fatalf("last_seen = %v, want the later observation %v", snapshot["h1"].LastSeen, later)
	}
}

func TestObserve_IgnoresEmptyHash(t *testing.T) {
	aggregator := NewAggregator()
	aggregator.Observe(Record{Model: "m", Detail: Detail{Total: 5}})

	snapshot := aggregator.Snapshot()
	if len(snapshot) != 0 {
		t.Fatalf("snapshot = %+v, want nothing recorded", snapshot)
	}
}

func TestSeed_ReplacesAndIsolatesState(t *testing.T) {
	aggregator := NewAggregator()
	aggregator.Observe(Record{Hash: "stale", Model: "m"})

	seeded := map[string]KeyUsage{
		"h1": {Counters: Counters{Requests: 4, Total: 40}, Models: map[string]ModelUsage{"m": {Counters: Counters{Requests: 4}}}},
	}
	aggregator.Seed(seeded)

	snapshot := aggregator.Snapshot()
	if len(snapshot) != 1 || snapshot["h1"].Requests != 4 {
		t.Fatalf("snapshot = %+v, want only the seeded entry", snapshot)
	}
	if aggregator.Dirty() {
		t.Fatal("Dirty() = true right after Seed, want false")
	}

	seeded["h1"].Models["m"] = ModelUsage{}
	aggregator.Observe(Record{Hash: "h1", Model: "m2"})
	if _, ok := snapshot["h1"].Models["m2"]; ok {
		t.Fatal("snapshot changed after later observations")
	}
}

func TestDirty_TracksUnpersistedChanges(t *testing.T) {
	aggregator := NewAggregator()
	if aggregator.Dirty() {
		t.Fatal("fresh aggregator reports dirty")
	}
	aggregator.Observe(Record{Hash: "h1", Model: "m"})
	if !aggregator.Dirty() {
		t.Fatal("Dirty() = false after an observation")
	}
	aggregator.MarkClean()
	if aggregator.Dirty() {
		t.Fatal("Dirty() = true after MarkClean")
	}
}

func TestObserve_IsConcurrencySafe(t *testing.T) {
	aggregator := NewAggregator()
	const (
		workers = 8
		perWork = 50
	)

	var waitGroup sync.WaitGroup
	for worker := range workers {
		waitGroup.Add(1)
		go func(worker int) {
			defer waitGroup.Done()
			for range perWork {
				aggregator.Observe(Record{Hash: "h1", Model: "m", Detail: Detail{Total: 1}, At: time.Now()})
			}
		}(worker)
	}
	waitGroup.Wait()

	snapshot := aggregator.Snapshot()
	if got := snapshot["h1"].Requests; got != workers*perWork {
		t.Fatalf("requests = %d, want %d", got, workers*perWork)
	}
	if got := snapshot["h1"].Models["m"].Requests; got != workers*perWork {
		t.Fatalf("model requests = %d, want %d", got, workers*perWork)
	}
}
