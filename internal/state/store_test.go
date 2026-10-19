package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cpa-secret-manager/internal/quota"
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

func TestStore_RetentionDaysBounds(t *testing.T) {
	store := NewDocument()
	if store.AppConfig.RetentionDays != DefaultRetentionDays {
		t.Fatalf("default retention days = %d, want %d", store.AppConfig.RetentionDays, DefaultRetentionDays)
	}

	s, _ := Load(filepath.Join(t.TempDir(), "state.json"))
	s.SetAppConfig(AppConfig{UsageEnabled: true, RetentionDays: 0})
	if s.AppConfig().RetentionDays != DefaultRetentionDays {
		t.Fatalf("retention days 0 should clamp to default, got %d", s.AppConfig().RetentionDays)
	}
	s.SetAppConfig(AppConfig{UsageEnabled: true, RetentionDays: 999})
	if s.AppConfig().RetentionDays != MaxRetentionDays {
		t.Fatalf("retention days 999 should clamp to max %d, got %d", MaxRetentionDays, s.AppConfig().RetentionDays)
	}
	s.SetAppConfig(AppConfig{UsageEnabled: true, RetentionDays: 15})
	if s.AppConfig().RetentionDays != 15 {
		t.Fatalf("retention days 15 should be kept, got %d", s.AppConfig().RetentionDays)
	}
}

func TestQuota_RoundTripAndIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	rules := map[string]quota.Rule{
		"h1": {
			Hash:       "h1",
			Enabled:    true,
			Strategy:   quota.StrategyTypeToken,
			Window:     quota.WindowTypeDaily,
			LimitValue: 5000,
			UpdatedAt:  now,
		},
	}
	states := map[string]quota.WindowState{
		"h1": {
			Hash:        "h1",
			WindowStart: now,
			WindowEnd:   now.AddDate(0, 0, 1),
			UsedValue:   1200,
		},
	}

	store.ReplaceQuotaRules(rules)
	store.ReplaceQuotaStates(states)
	if err := store.SaveAtomic(); err != nil {
		t.Fatalf("SaveAtomic() error = %v", err)
	}

	// Reload and verify
	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reloaded Load() error = %v", err)
	}
	loadedRules := reloaded.QuotaRules()
	if len(loadedRules) != 1 || loadedRules["h1"].LimitValue != 5000 {
		t.Fatalf("loaded rules mismatch: %+v", loadedRules)
	}
	loadedStates := reloaded.QuotaStates()
	if len(loadedStates) != 1 || loadedStates["h1"].UsedValue != 1200 {
		t.Fatalf("loaded states mismatch: %+v", loadedStates)
	}
}
