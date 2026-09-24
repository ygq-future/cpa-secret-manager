package state

import (
	"os"
	"path/filepath"
	"testing"

	"cpa-secret-manager/internal/usage"
)

func TestLoad_MissingFileYieldsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")

	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for a missing file", err)
	}
	if got := store.AppConfig(); got != DefaultAppConfig() {
		t.Fatalf("AppConfig() = %+v, want %+v", got, DefaultAppConfig())
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("Load() created %s on read; want lazy creation", path)
	}
}

func TestSaveAtomic_RoundTripsDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "state.json")

	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	store.SetAppConfig(AppConfig{UsageEnabled: false})
	store.ReplaceUsage(map[string]usage.KeyUsage{
		"hash-a": {Counters: usage.Counters{Requests: 2, Total: 30}, Models: map[string]usage.ModelUsage{"m": {Counters: usage.Counters{Requests: 2}}}},
	})
	if err := store.SaveAtomic(); err != nil {
		t.Fatalf("SaveAtomic() error = %v", err)
	}

	reloaded, reloadErr := Load(path)
	if reloadErr != nil {
		t.Fatalf("reload error = %v", reloadErr)
	}
	if got := reloaded.AppConfig(); got.UsageEnabled {
		t.Fatal("reloaded usage_enabled = true, want false")
	}
	entry, ok := reloaded.Usage()["hash-a"]
	if !ok || entry.Requests != 2 || entry.Models["m"].Requests != 2 {
		t.Fatalf("reloaded usage = %+v (ok=%v), want the persisted counters", entry, ok)
	}
	if got := reloaded.Snapshot().SchemaVersion; got != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", got, SchemaVersion)
	}
	if _, statErr := os.Stat(path + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatalf("temporary file %s.tmp was left behind", path)
	}
}

func TestLoad_CorruptFileKeepsServiceUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want a decode error for a corrupt document")
	}
	if got := store.AppConfig(); got != DefaultAppConfig() {
		t.Fatalf("AppConfig() = %+v, want defaults after a corrupt document", got)
	}
}

func TestUsage_IsIsolatedFromLaterMutations(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	store.ReplaceUsage(map[string]usage.KeyUsage{
		"hash-a": {Counters: usage.Counters{Requests: 1}, Models: map[string]usage.ModelUsage{"m": {}}},
	})

	snapshot := store.Snapshot()
	snapshot.Usage["hash-a"].Models["injected"] = usage.ModelUsage{}

	fresh := store.Usage()
	if _, ok := fresh["hash-a"].Models["injected"]; ok {
		t.Fatal("Usage() returned a map that aliases internal state")
	}
}

func TestReplaceUsage_IgnoresNilStore(t *testing.T) {
	var store *Store
	store.ReplaceUsage(map[string]usage.KeyUsage{"hash-a": {}})
	if got := store.Usage(); len(got) != 0 {
		t.Fatalf("Usage() = %+v, want an empty map", got)
	}
}

func TestSnapshot_IsIsolatedFromLaterMutations(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	snapshot := store.Snapshot()
	store.SetAppConfig(AppConfig{UsageEnabled: false})

	if !snapshot.AppConfig.UsageEnabled {
		t.Fatal("snapshot changed after the store was mutated")
	}
}
