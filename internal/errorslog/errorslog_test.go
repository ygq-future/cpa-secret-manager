package errorslog_test

import (
	"path/filepath"
	"testing"
	"time"

	"cpa-secret-manager/internal/errorslog"
)

func TestErrorsLogRecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "errors.json")

	store, err := errorslog.Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	t0 := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Hour)
	t2 := t0.Add(2 * time.Hour)

	store.Record("hash1", errorslog.ErrorEntry{
		Time:       t0,
		Model:      "gpt-5.6",
		StatusCode: 429,
		Message:    "rate limit exceeded",
	})
	store.Record("hash1", errorslog.ErrorEntry{
		Time:       t2,
		Model:      "gpt-5.6",
		StatusCode: 500,
		Message:    "internal error",
	})
	store.Record("hash2", errorslog.ErrorEntry{
		Time:       t1,
		Model:      "claude-3-7-sonnet",
		StatusCode: 503,
		Message:    "service unavailable",
	})

	if !store.Dirty() {
		t.Fatal("expected store to be dirty")
	}

	// Query hash1 without time range
	res := store.Query("hash1", time.Time{}, time.Time{})
	if len(res) != 2 {
		t.Fatalf("expected 2 entries for hash1, got %d", len(res))
	}
	// Verify descending order
	if res[0].StatusCode != 500 || res[1].StatusCode != 429 {
		t.Fatalf("expected newest first, got %+v", res)
	}

	// Query with time range [t0 + 30m, t0 + 90m)
	resRange := store.Query("hash2", t0.Add(30*time.Minute), t0.Add(90*time.Minute))
	if len(resRange) != 1 || resRange[0].StatusCode != 503 {
		t.Fatalf("expected 1 entry for hash2 in range, got %+v", resRange)
	}

	// Save and reload
	if err := store.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if store.Dirty() {
		t.Fatal("expected store not dirty after save")
	}

	loaded, err := errorslog.Load(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	all := loaded.QueryAll(time.Time{}, time.Time{})
	if len(all["hash1"]) != 2 || len(all["hash2"]) != 1 {
		t.Fatalf("unexpected loaded contents: %+v", all)
	}

	// Prune older than t1
	removed := loaded.Prune(t1)
	if removed != 1 {
		t.Fatalf("expected 1 entry pruned, got %d", removed)
	}
	left := loaded.Query("hash1", time.Time{}, time.Time{})
	if len(left) != 1 || left[0].StatusCode != 500 {
		t.Fatalf("expected only t2 left for hash1, got %+v", left)
	}

	// Forget hash1
	if !loaded.Forget("hash1") {
		t.Fatal("expected Forget(hash1) to return true")
	}
	if len(loaded.Query("hash1", time.Time{}, time.Time{})) != 0 {
		t.Fatal("expected hash1 to be empty after Forget")
	}
}
