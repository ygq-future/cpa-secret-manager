package state

import (
	"os"
	"path/filepath"
	"testing"

	"cpa-secret-manager/internal/pricing"
)

func TestPriceStore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prices.json")
	store, err := LoadPrices(path)
	if err != nil {
		t.Fatalf("LoadPrices() error = %v", err)
	}
	if len(store.Suggestions()) != 0 {
		t.Fatalf("new store suggestions = %+v, want empty", store.Suggestions())
	}

	store.Replace("models.dev", "2026-11-10T12:00:00Z", `"abc"`, map[string]pricing.Suggestion{
		"gpt-4o": {Model: "gpt-4o", Provider: "openai", Input: 2.5, Output: 10, Cache: 1.25, Providers: 3},
	}, []string{"gpt-4o", "missing"})
	if err := store.SaveAtomic(); err != nil {
		t.Fatalf("SaveAtomic() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file: %v", err)
	}

	reloaded, err := LoadPrices(path)
	if err != nil {
		t.Fatalf("LoadPrices() error = %v", err)
	}
	source, fetchedAt, etag := reloaded.Meta()
	if source != "models.dev" || fetchedAt != "2026-11-10T12:00:00Z" || etag != `"abc"` {
		t.Fatalf("meta = %q/%q/%q, want models.dev/2026-11-10T12:00:00Z/\"abc\"", source, fetchedAt, etag)
	}
	got := reloaded.Suggestions()["gpt-4o"]
	if got.Provider != "openai" || got.Input != 2.5 || got.Output != 10 || got.Cache != 1.25 {
		t.Fatalf("reloaded suggestion = %+v", got)
	}
	known := reloaded.Known()
	if len(known) != 2 || known[0] != "gpt-4o" || known[1] != "missing" {
		t.Fatalf("known models = %#v, want the requested set including misses", known)
	}
}

func TestPriceStore_NormalizesKnownModels(t *testing.T) {
	store, err := LoadPrices(filepath.Join(t.TempDir(), "prices.json"))
	if err != nil {
		t.Fatalf("LoadPrices() error = %v", err)
	}
	store.Replace("models.dev", "now", "", nil, []string{"GPT-4o", "gpt-4o", "Claude-Sonnet-4-5"})
	known := store.Known()
	if len(known) != 2 || known[0] != "gpt-4o" || known[1] != "claude-sonnet-4-5" {
		t.Fatalf("known models = %#v, want normalized deduplicated names", known)
	}
}

func TestPriceStore_ReplaceCopiesInput(t *testing.T) {
	store, err := LoadPrices(filepath.Join(t.TempDir(), "prices.json"))
	if err != nil {
		t.Fatalf("LoadPrices() error = %v", err)
	}
	source := map[string]pricing.Suggestion{
		"gpt-4o": {Input: 1},
	}
	store.Replace("models.dev", "now", "", source, []string{"gpt-4o"})
	source["gpt-4o"] = pricing.Suggestion{Input: 99}
	if got := store.Suggestions()["gpt-4o"].Input; got != 1 {
		t.Fatalf("stored input = %v, want the value captured at Replace time", got)
	}

	snapshot := store.Snapshot()
	snapshot.Models["gpt-4o"] = pricing.Suggestion{Input: 42}
	if got := store.Suggestions()["gpt-4o"].Input; got != 1 {
		t.Fatalf("snapshot mutation leaked into the store: input = %v", got)
	}
}

func TestPriceStore_MalformedFileKeepsService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prices.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	store, err := LoadPrices(path)
	if err == nil {
		t.Fatal("LoadPrices() error = nil for malformed cache")
	}
	if store == nil {
		t.Fatal("LoadPrices() store = nil, want an empty usable store")
	}
	if len(store.Suggestions()) != 0 {
		t.Fatalf("suggestions = %+v, want empty after a malformed cache", store.Suggestions())
	}
}

func TestPriceStore_EmptyPath(t *testing.T) {
	store, err := LoadPrices("")
	if err == nil {
		t.Fatal("LoadPrices() error = nil for an empty path")
	}
	if store == nil {
		t.Fatal("LoadPrices() store = nil, want a usable empty store")
	}
}
