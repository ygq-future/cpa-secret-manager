package errorslog_test

import (
	"os"
	"path/filepath"
	"strings"
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
		AuthID:     "codex-user@example.com.json",
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
	if all["hash1"][0].AuthID != "codex-user@example.com.json" {
		t.Fatalf("auth id did not survive reload: %+v", all["hash1"][0])
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

// TestErrorsLogDecodesLegacyEscapedMessages pins the schema bump: documents
// written before plain-text storage keep the HTML-escaped host body, and loading
// decodes each message exactly once. The nested entity proves the reload does
// not decode a second time.
func TestErrorsLogDecodesLegacyEscapedMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "errors.json")
	legacy := `{"schema_version":1,"failures":{"hash1":[{"time":"2026-11-16T08:00:00Z","model":"gemini-3.5-flash-lite","status_code":429,"message":"body &amp;#34;quoted&amp;#34;"}]}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy document: %v", err)
	}

	store, err := errorslog.Load(path)
	if err != nil {
		t.Fatalf("load legacy document: %v", err)
	}
	entries := store.Query("hash1", time.Time{}, time.Time{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 legacy entry, got %d", len(entries))
	}
	if got, want := entries[0].Message, `body &#34;quoted&#34;`; got != want {
		t.Fatalf("legacy message decoded to %q, want %q", got, want)
	}
	if !store.Dirty() {
		t.Fatal("a migrated document must be dirty so the decoded text is persisted")
	}
	if err := store.Save(); err != nil {
		t.Fatalf("save migrated document: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migrated document: %v", err)
	}
	if !strings.Contains(string(raw), `"schema_version":2`) {
		t.Fatalf("migrated document should carry schema version 2: %s", raw)
	}

	reloaded, err := errorslog.Load(path)
	if err != nil {
		t.Fatalf("reload migrated document: %v", err)
	}
	again := reloaded.Query("hash1", time.Time{}, time.Time{})
	if len(again) != 1 || again[0].Message != `body &#34;quoted&#34;` {
		t.Fatalf("reload decoded the message twice: %+v", again)
	}
	if reloaded.Dirty() {
		t.Fatal("a current document must not be rewritten on load")
	}
}

func TestErrorsLogClearBefore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "errors.json")
	store, err := errorslog.Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	t0 := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Hour)
	t2 := t0.Add(2 * time.Hour)

	store.Record("hash1", errorslog.ErrorEntry{Time: t0, Model: "gpt-5.6", AuthID: "auth-a", StatusCode: 0, Message: "old zero"})
	store.Record("hash1", errorslog.ErrorEntry{Time: t0, Model: "gpt-5.6", AuthID: "auth-a", StatusCode: 429, Message: "old 429"})
	store.Record("hash1", errorslog.ErrorEntry{Time: t1, Model: "gpt-5.6", AuthID: "auth-a", StatusCode: 500, Message: "exact cutoff"})
	store.Record("hash2", errorslog.ErrorEntry{Time: t0, Model: "gpt-5.6", AuthID: "auth-a", StatusCode: 503, Message: "other key"})
	store.Record("hash2", errorslog.ErrorEntry{Time: t2, Model: "claude-3-7-sonnet", AuthID: "auth-b", StatusCode: 500, Message: "newer"})
	if err := store.Save(); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	// Zero Before must be a no-op.
	if got := store.ClearBefore(errorslog.ClearFilter{}); got != 0 || store.Dirty() {
		t.Fatalf("zero Before removed %d (dirty=%v), want 0 and clean", got, store.Dirty())
	}

	// Filtered clear for status_code == 0 before t1 removes only the zero-status entry.
	zero := 0
	if got := store.ClearBefore(errorslog.ClearFilter{Before: t1, Hash: "hash1", StatusCode: &zero}); got != 1 {
		t.Fatalf("status_code=0 clear removed %d, want 1", got)
	}
	if !store.Dirty() {
		t.Fatal("expected store dirty after ClearBefore")
	}

	// Filtered clear by model across all hashes before t1 removes hash1@429 and hash2@503, keeping hash1@t1 and hash2@t2.
	if got := store.ClearBefore(errorslog.ClearFilter{Before: t1, Model: "gpt-5.6"}); got != 2 {
		t.Fatalf("model clear removed %d, want 2", got)
	}

	leftHash1 := store.Query("hash1", time.Time{}, time.Time{})
	if len(leftHash1) != 1 || !leftHash1[0].Time.Equal(t1) {
		t.Fatalf("hash1 entries = %+v, want only t1 entry", leftHash1)
	}
	leftHash2 := store.Query("hash2", time.Time{}, time.Time{})
	if len(leftHash2) != 1 || !leftHash2[0].Time.Equal(t2) {
		t.Fatalf("hash2 entries = %+v, want only t2 entry", leftHash2)
	}
}
