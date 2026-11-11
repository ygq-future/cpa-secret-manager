package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateLayout_OldDefaultFilesMoveIntoDataDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	oldState := filepath.Join("data", "cpa-secret-manager-cache.json")
	oldFiles := map[string]string{
		oldState:                             "state",
		oldSiblingPath(oldState, "-buckets"): "buckets",
		oldSiblingPath(oldState, "-errors"):  "errors",
		oldSiblingPath(oldState, "-prices"):  "prices",
	}
	for path, content := range oldFiles {
		writeTestFile(t, path, content)
	}

	newState, warnings := MigrateLayout(oldState, "data/cpa-secret-manager/state.json")
	if len(warnings) != 0 {
		t.Fatalf("MigrateLayout() warnings = %v", warnings)
	}
	if filepath.Clean(newState) != filepath.Clean("data/cpa-secret-manager/state.json") {
		t.Fatalf("MigrateLayout() state = %q", newState)
	}

	newFiles := map[string]string{
		newState: "state",
		filepath.Join("data", "cpa-secret-manager", "buckets.json"): "buckets",
		filepath.Join("data", "cpa-secret-manager", "errors.json"):  "errors",
		filepath.Join("data", "cpa-secret-manager", "prices.json"):  "prices",
	}
	for path, want := range newFiles {
		if got := readTestFile(t, path); got != want {
			t.Fatalf("%s content = %q, want %q", path, got, want)
		}
	}
	for path := range oldFiles {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("old file %s still exists", path)
		}
	}

	again, warnings := MigrateLayout(newState, "data/cpa-secret-manager/state.json")
	if len(warnings) != 0 || filepath.Clean(again) != filepath.Clean(newState) {
		t.Fatalf("second MigrateLayout() = %q, %v; want idempotent %q", again, warnings, newState)
	}
}

func TestMigrateLayout_NewDefaultPathMigratesOldSiblings(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	newState := filepath.Join("data", "cpa-secret-manager", "state.json")
	oldState := filepath.Join("data", "cpa-secret-manager-cache.json")
	writeTestFile(t, oldState, "old-state")
	writeTestFile(t, oldSiblingPath(oldState, "-prices"), "old-prices")

	migrated, warnings := MigrateLayout(newState, "data/cpa-secret-manager/state.json")
	if len(warnings) != 0 {
		t.Fatalf("MigrateLayout() warnings = %v", warnings)
	}
	if filepath.Clean(migrated) != filepath.Clean(newState) {
		t.Fatalf("MigrateLayout() state = %q", migrated)
	}
	if got := readTestFile(t, newState); got != "old-state" {
		t.Fatalf("migrated state = %q", got)
	}
	if got := readTestFile(t, filepath.Join("data", "cpa-secret-manager", "prices.json")); got != "old-prices" {
		t.Fatalf("migrated prices = %q", got)
	}
}

func TestMigrateLayout_CustomPathKeepsStateAndMovesSiblings(t *testing.T) {
	dir := t.TempDir()
	customState := filepath.Join(dir, "custom", "state.json")
	writeTestFile(t, customState, "custom-state")
	writeTestFile(t, oldSiblingPath(customState, "-buckets"), "custom-buckets")

	migrated, warnings := MigrateLayout(customState, filepath.Join(dir, "default", "state.json"))
	if len(warnings) != 0 {
		t.Fatalf("MigrateLayout() warnings = %v", warnings)
	}
	if filepath.Clean(migrated) != filepath.Clean(customState) {
		t.Fatalf("MigrateLayout() state = %q, want the custom path", migrated)
	}
	if got := readTestFile(t, customState); got != "custom-state" {
		t.Fatalf("custom state changed to %q", got)
	}
	if got := readTestFile(t, filepath.Join(dir, "custom", "buckets.json")); got != "custom-buckets" {
		t.Fatalf("migrated custom buckets = %q", got)
	}
	if _, err := os.Stat(oldSiblingPath(customState, "-buckets")); !os.IsNotExist(err) {
		t.Fatal("old custom bucket file still exists")
	}
}

func TestMigrateLayout_ExistingDestinationWins(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	oldState := filepath.Join("data", "cpa-secret-manager-cache.json")
	newState := filepath.Join("data", "cpa-secret-manager", "state.json")
	writeTestFile(t, oldState, "old-state")
	writeTestFile(t, newState, "new-state")

	migrated, warnings := MigrateLayout(oldState, "data/cpa-secret-manager/state.json")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "was not removed") {
		t.Fatalf("MigrateLayout() warnings = %v, want one destination-exists warning", warnings)
	}
	if filepath.Clean(migrated) != filepath.Clean(newState) {
		t.Fatalf("MigrateLayout() state = %q", migrated)
	}
	if got := readTestFile(t, newState); got != "new-state" {
		t.Fatalf("existing destination was overwritten: %q", got)
	}
	if _, err := os.Stat(oldState); err != nil {
		t.Fatalf("old state should be left in place when the destination exists: %v", err)
	}
}

func TestMigrateLayout_MissingFilesAreNoOp(t *testing.T) {
	dir := t.TempDir()
	customState := filepath.Join(dir, "custom", "state.json")
	migrated, warnings := MigrateLayout(customState, filepath.Join(dir, "default", "state.json"))
	if len(warnings) != 0 {
		t.Fatalf("MigrateLayout() warnings = %v", warnings)
	}
	if filepath.Clean(migrated) != filepath.Clean(customState) {
		t.Fatalf("MigrateLayout() state = %q", migrated)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
