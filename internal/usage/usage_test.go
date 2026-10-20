package usage

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestObserve_SplitsByHourAndModel(t *testing.T) {
	aggregator := NewAggregator()
	base := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

	aggregator.Observe(Record{Hash: "h1", Model: "gpt-5.6", At: base,
		Detail: Detail{Input: 10, Output: 20, Reasoning: 3, Cached: 4, Total: 33}})
	aggregator.Observe(Record{Hash: "h1", Model: "gpt-5.6", Failed: true, At: base.Add(time.Minute),
		Detail: Detail{Input: 1, Output: 2, Total: 3}})
	aggregator.Observe(Record{Hash: "h1", Model: "claude-sonnet", At: base.Add(2 * time.Minute),
		Detail: Detail{Input: 5, CacheRead: 7, CacheCreation: 9, Total: 21}})
	aggregator.Observe(Record{Hash: "h1", Model: "claude-sonnet", At: base.Add(time.Hour),
		Detail: Detail{Total: 7}})

	buckets := aggregator.SnapshotBuckets()["h1"]
	if len(buckets) != 2 {
		t.Fatalf("buckets = %d, want one per hour", len(buckets))
	}
	first := buckets[PeriodKey(base)]
	want := Counters{Requests: 3, Failed: 1, Input: 16, Output: 22, Reasoning: 3, Cached: 4, CacheRead: 7, CacheCreation: 9, Total: 57}
	if first.Counters != want {
		t.Fatalf("hour counters = %+v, want %+v", first.Counters, want)
	}
	if got := first.Models["gpt-5.6"]; got.Requests != 2 || got.Failed != 1 || got.Total != 36 {
		t.Fatalf("gpt-5.6 counters = %+v, want 2 requests (1 failed) and 36 total tokens", got)
	}
	second := buckets[PeriodKey(base.Add(time.Hour))]
	if second.Requests != 1 || second.Models["claude-sonnet"].Total != 7 {
		t.Fatalf("next hour bucket = %+v, want the single claude record", second)
	}
}

func TestRange_MergesOnlyOverlappingBuckets(t *testing.T) {
	aggregator := NewAggregator()
	day := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

	aggregator.Observe(Record{Hash: "h1", Model: "m", At: day, Detail: Detail{Total: 10}})
	aggregator.Observe(Record{Hash: "h1", Model: "m", At: day.AddDate(0, 0, -1), Detail: Detail{Total: 1}})
	aggregator.Observe(Record{Hash: "h2", Model: "m", At: day, Detail: Detail{Total: 5}})

	dayStart := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)
	got := aggregator.Range(dayStart, dayEnd)

	if len(got) != 2 {
		t.Fatalf("keys = %d, want the two observed keys", len(got))
	}
	if got["h1"].Total != 10 || got["h1"].Requests != 1 {
		t.Fatalf("h1 = %+v, want only the bucket inside the window", got["h1"].Counters)
	}
	if got["h2"].Total != 5 {
		t.Fatalf("h2 = %+v, want its single in-window request", got["h2"].Counters)
	}
	if got["h1"].Models["m"].Total != 10 {
		t.Fatalf("h1 model split = %+v, want the same window", got["h1"].Models)
	}
	if !got["h1"].FirstSeen.Equal(day) || !got["h1"].LastSeen.Equal(day) {
		t.Fatalf("window = %v..%v, want the exact observed instant %v", got["h1"].FirstSeen, got["h1"].LastSeen, day)
	}
}

func TestBucket_ZeroLastAtIsOmitted(t *testing.T) {
	legacy, err := json.Marshal(Bucket{Counters: Counters{Requests: 1}})
	if err != nil {
		t.Fatalf("marshal legacy bucket: %v", err)
	}
	if strings.Contains(string(legacy), "last_at") {
		t.Fatalf("legacy bucket = %s, want no last_at field", legacy)
	}

	at := time.Date(2026, 9, 23, 10, 42, 7, 0, time.UTC)
	observed, err := json.Marshal(Bucket{Counters: Counters{Requests: 1}, LastAt: at})
	if err != nil {
		t.Fatalf("marshal observed bucket: %v", err)
	}
	if !strings.Contains(string(observed), `"last_at":"2026-09-23T10:42:07Z"`) {
		t.Fatalf("observed bucket = %s, want the precise last_at", observed)
	}
}

func TestLastSeen_ReportsLatestObservedInstant(t *testing.T) {
	aggregator := NewAggregator()
	base := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	latest := base.Add(47 * time.Minute)

	aggregator.Observe(Record{Hash: "h1", Model: "m", At: base, Detail: Detail{Total: 1}})
	aggregator.Observe(Record{Hash: "h1", Model: "m", At: latest, Detail: Detail{Total: 2}})

	if got := aggregator.LastSeen()["h1"]; !got.Equal(latest) {
		t.Fatalf("last seen = %v, want the exact latest record %v", got, latest)
	}

	// A bucket written before precise stamps existed reports its start: a lower
	// bound that cannot claim a call in the future.
	legacy := map[string]map[string]Bucket{
		"h2": {PeriodKey(base): {Counters: Counters{Requests: 1}}},
	}
	aggregator.SeedBuckets(legacy)
	if got := aggregator.LastSeen()["h2"]; !got.Equal(base) {
		t.Fatalf("legacy last seen = %v, want the bucket start %v", got, base)
	}
}

func TestRange_EmptyWindowReturnsNothing(t *testing.T) {
	aggregator := NewAggregator()
	at := time.Date(2026, 9, 23, 10, 30, 0, 0, time.UTC)
	aggregator.Observe(Record{Hash: "h1", Model: "m", At: at})

	if got := aggregator.Range(at, at); len(got) != 0 {
		t.Fatalf("zero-width window = %+v, want nothing", got)
	}
	if got := aggregator.Range(at.Add(time.Hour), at.Add(2*time.Hour)); len(got) != 0 {
		t.Fatalf("future window = %+v, want nothing", got)
	}
}

func TestPrune_DropsBucketsOutsideRetention(t *testing.T) {
	aggregator := NewAggregator()
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	aggregator.Observe(Record{Hash: "old", Model: "m", At: now.AddDate(0, 0, -40)})
	aggregator.Observe(Record{Hash: "recent", Model: "m", At: now.AddDate(0, 0, -2)})
	aggregator.MarkClean()

	if removed := aggregator.Prune(now.AddDate(0, 0, -30)); removed != 1 {
		t.Fatalf("Prune removed %d buckets, want 1", removed)
	}
	buckets := aggregator.SnapshotBuckets()
	if _, ok := buckets["old"]; ok {
		t.Fatal("expired key survived Prune")
	}
	if _, ok := buckets["recent"]; !ok {
		t.Fatal("in-retention key was pruned")
	}
	if !aggregator.Dirty() {
		t.Fatal("Dirty() = false after Prune dropped data, want a pending write")
	}
	if removed := aggregator.Prune(time.Time{}); removed != 0 {
		t.Fatalf("Prune(zero) removed %d buckets, want 0", removed)
	}
}

func TestObserve_GroupsMissingModelAsUnknown(t *testing.T) {
	aggregator := NewAggregator()
	aggregator.Observe(Record{Hash: "h1", Detail: Detail{Total: 1}})

	bucket := aggregator.SnapshotBuckets()["h1"][PeriodKey(time.Now())]
	if _, ok := bucket.Models[UnknownModel]; !ok {
		t.Fatalf("models = %+v, want an %s bucket", bucket.Models, UnknownModel)
	}
}

func TestObserve_IgnoresEmptyHash(t *testing.T) {
	aggregator := NewAggregator()
	aggregator.Observe(Record{Model: "m", Detail: Detail{Total: 5}})

	if buckets := aggregator.SnapshotBuckets(); len(buckets) != 0 {
		t.Fatalf("buckets = %+v, want nothing recorded", buckets)
	}
}

func TestSeedBuckets_ReplacesAndIsolatesState(t *testing.T) {
	aggregator := NewAggregator()
	aggregator.Observe(Record{Hash: "stale", Model: "m"})

	period := PeriodKey(time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC))
	lastAt := time.Date(2026, 9, 23, 10, 42, 7, 0, time.UTC)
	seeded := map[string]map[string]Bucket{
		"h1": {period: {Counters: Counters{Requests: 4, Total: 40}, Models: map[string]Counters{"m": {Requests: 4}}, LastAt: lastAt}},
	}
	aggregator.SeedBuckets(seeded)

	snapshot := aggregator.SnapshotBuckets()
	if len(snapshot) != 1 || snapshot["h1"][period].Requests != 4 {
		t.Fatalf("snapshot = %+v, want only the seeded bucket", snapshot)
	}
	if !snapshot["h1"][period].LastAt.Equal(lastAt) {
		t.Fatalf("snapshot last-at = %v, want %v", snapshot["h1"][period].LastAt, lastAt)
	}
	if aggregator.Dirty() {
		t.Fatal("Dirty() = true right after SeedBuckets, want false")
	}

	seeded["h1"][period] = Bucket{}
	aggregator.Observe(Record{Hash: "h1", Model: "m2", At: time.Date(2026, 9, 23, 10, 5, 0, 0, time.UTC)})
	if snapshot["h1"][period].Models["m2"].Requests != 0 {
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

func TestRemove_DropsEveryBucketOfOneKey(t *testing.T) {
	aggregator := NewAggregator()
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	aggregator.Observe(Record{Hash: "h1", Model: "m", At: at})
	aggregator.Observe(Record{Hash: "h1", Model: "m", At: at.Add(time.Hour)})
	aggregator.Observe(Record{Hash: "h2", Model: "m", At: at})

	if !aggregator.Remove("h1") {
		t.Fatal("Remove(h1) = false, want true")
	}
	if aggregator.Remove("h1") {
		t.Fatal("Remove(h1) on the second call = true, want false")
	}
	buckets := aggregator.SnapshotBuckets()
	if _, ok := buckets["h1"]; ok {
		t.Fatal("h1 buckets survived Remove")
	}
	if _, ok := buckets["h2"]; !ok {
		t.Fatal("Remove(h1) also dropped h2")
	}
}

func TestHashesAndModels_ReportObservedValues(t *testing.T) {
	aggregator := NewAggregator()
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	aggregator.Observe(Record{Hash: "h1", Model: "a", At: at})
	aggregator.Observe(Record{Hash: "h2", Model: "b", At: at})

	hashes := aggregator.Hashes()
	if len(hashes) != 2 {
		t.Fatalf("hashes = %v, want both keys", hashes)
	}
	models := aggregator.Models()
	if len(models) != 2 {
		t.Fatalf("models = %v, want a and b", models)
	}
}

func TestObserve_IsConcurrencySafe(t *testing.T) {
	aggregator := NewAggregator()
	const (
		workers = 8
		perWork = 50
	)
	now := time.Now().UTC()

	var waitGroup sync.WaitGroup
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for range perWork {
				aggregator.Observe(Record{Hash: "h1", Model: "m", Detail: Detail{Total: 1}, At: now})
			}
		}()
	}
	waitGroup.Wait()

	got := aggregator.Range(now.Add(-time.Hour), now.Add(time.Hour))["h1"]
	if got.Requests != workers*perWork {
		t.Fatalf("requests = %d, want %d", got.Requests, workers*perWork)
	}
	if got.Models["m"].Requests != workers*perWork {
		t.Fatalf("model requests = %d, want %d", got.Models["m"].Requests, workers*perWork)
	}
}
