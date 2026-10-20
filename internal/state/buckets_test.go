package state

import (
	"os"
	"path/filepath"
	"testing"

	"cpa-secret-manager/internal/usage"
)

func sampleBuckets() map[string]map[string]usage.Bucket {
	return map[string]map[string]usage.Bucket{
		"hash-a": {
			"2026-09-23T10": {
				Counters: usage.Counters{Requests: 2, Failed: 1, Total: 30},
				Models:   map[string]usage.Counters{"gpt-5.6": {Requests: 2, Total: 30}},
			},
		},
	}
}

func TestLoadBuckets_MissingFileYieldsEmptyDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "buckets.json")

	store, err := LoadBuckets(path)
	if err != nil {
		t.Fatalf("LoadBuckets() error = %v, want nil for a missing file", err)
	}
	if got := store.Buckets(); len(got) != 0 {
		t.Fatalf("Buckets() = %+v, want empty", got)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("LoadBuckets() created %s on read; want lazy creation", path)
	}
}

func TestSaveAtomic_RoundTripsBuckets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "buckets.json")

	store, err := LoadBuckets(path)
	if err != nil {
		t.Fatalf("LoadBuckets() error = %v", err)
	}
	store.ReplaceBuckets(sampleBuckets())
	if err := store.SaveAtomic(); err != nil {
		t.Fatalf("SaveAtomic() error = %v", err)
	}

	reloaded, reloadErr := LoadBuckets(path)
	if reloadErr != nil {
		t.Fatalf("reload error = %v", reloadErr)
	}
	entry := reloaded.Buckets()["hash-a"]["2026-09-23T10"]
	if entry.Requests != 2 || entry.Failed != 1 || entry.Total != 30 {
		t.Fatalf("reloaded bucket = %+v, want the persisted counters", entry)
	}
	if entry.Models["gpt-5.6"].Requests != 2 {
		t.Fatalf("reloaded models = %+v, want the model split", entry.Models)
	}
	if got := reloaded.doc.SchemaVersion; got != BucketSchemaVersion {
		t.Fatalf("schema version = %d, want %d", got, BucketSchemaVersion)
	}
	if _, statErr := os.Stat(path + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatalf("temporary file %s.tmp was left behind", path)
	}
}

func TestLoadBuckets_CorruptFileKeepsServiceUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buckets.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store, err := LoadBuckets(path)
	if err == nil {
		t.Fatal("LoadBuckets() error = nil, want a decode error for a corrupt document")
	}
	if got := store.Buckets(); len(got) != 0 {
		t.Fatalf("Buckets() = %+v, want an empty document after a corrupt file", got)
	}
}

func TestBuckets_AreIsolatedFromLaterMutations(t *testing.T) {
	store, err := LoadBuckets(filepath.Join(t.TempDir(), "buckets.json"))
	if err != nil {
		t.Fatalf("LoadBuckets() error = %v", err)
	}
	store.ReplaceBuckets(sampleBuckets())

	first := store.Buckets()
	first["hash-a"]["2026-09-23T10"].Models["injected"] = usage.Counters{}

	second := store.Buckets()
	if _, ok := second["hash-a"]["2026-09-23T10"].Models["injected"]; ok {
		t.Fatal("Buckets() returned a map that aliases internal state")
	}
}

func TestBucketBackup_WritesRecoverableCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buckets.json")
	store, err := LoadBuckets(path)
	if err != nil {
		t.Fatalf("LoadBuckets() error = %v", err)
	}
	store.ReplaceBuckets(sampleBuckets())
	if err := store.Backup(); err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	backup, backupErr := LoadBuckets(path + ".bak")
	if backupErr != nil {
		t.Fatalf("load backup error = %v", backupErr)
	}
	if got := backup.Buckets()["hash-a"]["2026-09-23T10"].Total; got != 30 {
		t.Fatalf("backup total = %d, want the pre-deletion counters", got)
	}
}

func TestBucketStore_NilAndEmptyPathAreSafe(t *testing.T) {
	var store *BucketStore
	store.ReplaceBuckets(sampleBuckets())
	if got := store.Buckets(); len(got) != 0 {
		t.Fatalf("Buckets() = %+v, want empty", got)
	}
	if err := store.SaveAtomic(); err == nil {
		t.Fatal("SaveAtomic() on a nil store = nil, want an error")
	}

	empty, err := LoadBuckets("")
	if err == nil {
		t.Fatal("LoadBuckets(\"\") = nil, want an error for an empty path")
	}
	if got := empty.Buckets(); len(got) != 0 {
		t.Fatalf("Buckets() = %+v, want empty", got)
	}
}
