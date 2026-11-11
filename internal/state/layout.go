package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// oldDefaultStatePath is the pre-directory layout. It is only used to
	// migrate existing installations into the plugin data directory.
	oldDefaultStatePath = "data/cpa-secret-manager-cache.json"
	bucketsFileName     = "buckets.json"
	errorsFileName      = "errors.json"
	pricesFileName      = "prices.json"
)

// MigrateLayout moves the old flat cache files into the plugin data directory.
// It is idempotent: a second call finds no old files and changes nothing.
//
// The returned path is the state document the runtime should use. If the old
// state document cannot be moved, the old path is returned so existing data is
// not silently orphaned; warnings describe what was left behind.
func MigrateLayout(statePath string, defaultStatePath string) (string, []string) {
	trimmed := strings.TrimSpace(statePath)
	if trimmed == "" {
		return "", nil
	}
	cleaned := filepath.Clean(trimmed)
	defaultPath := filepath.Clean(strings.TrimSpace(defaultStatePath))
	newState := cleaned

	var moves []fileMove
	switch {
	case isOldDefaultStatePath(cleaned):
		newState = filepath.Join(filepath.Dir(cleaned), "cpa-secret-manager", "state.json")
		moves = defaultLayoutMoves(newState)
	case cleaned == defaultPath:
		moves = defaultLayoutMoves(cleaned)
	default:
		dir := filepath.Dir(cleaned)
		moves = []fileMove{
			{from: oldSiblingPath(cleaned, "-buckets"), to: filepath.Join(dir, bucketsFileName)},
			{from: oldSiblingPath(cleaned, "-errors"), to: filepath.Join(dir, errorsFileName)},
			{from: oldSiblingPath(cleaned, "-prices"), to: filepath.Join(dir, pricesFileName)},
		}
	}

	var warnings []string
	for _, move := range moves {
		if filepath.Clean(move.from) == filepath.Clean(move.to) {
			continue
		}
		_, skipped, err := moveFile(move.from, move.to)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("state layout: %v", err))
			if filepath.Clean(move.to) == filepath.Clean(newState) {
				// Keep using the old state document when it still exists.
				if _, statErr := os.Stat(move.from); statErr == nil {
					return move.from, warnings
				}
			}
			continue
		}
		if skipped {
			warnings = append(warnings, fmt.Sprintf("state layout: kept %s; old file %s was not removed", move.to, move.from))
		}
	}
	return newState, warnings
}

// defaultLayoutMoves maps the four old default files into the new plugin data
// directory. The new state path is data/cpa-secret-manager/state.json, so the
// old default files live one directory above the new data directory.
func defaultLayoutMoves(newState string) []fileMove {
	newDir := filepath.Dir(newState)
	oldDir := filepath.Dir(newDir)
	oldState := filepath.Join(oldDir, "cpa-secret-manager-cache.json")
	return []fileMove{
		{from: oldState, to: newState},
		{from: oldSiblingPath(oldState, "-buckets"), to: filepath.Join(newDir, bucketsFileName)},
		{from: oldSiblingPath(oldState, "-errors"), to: filepath.Join(newDir, errorsFileName)},
		{from: oldSiblingPath(oldState, "-prices"), to: filepath.Join(newDir, pricesFileName)},
	}
}

type fileMove struct {
	from string
	to   string
}

func isOldDefaultStatePath(path string) bool {
	return filepath.Base(path) == filepath.Base(oldDefaultStatePath)
}

// moveFile renames one cache file, falling back to a copy-and-remove when the
// source and destination are on different filesystems. An existing destination
// is authoritative and is never overwritten; skipped reports that the old file
// was left in place.
func moveFile(from string, to string) (moved bool, skipped bool, err error) {
	if from == "" || to == "" || filepath.Clean(from) == filepath.Clean(to) {
		return false, false, nil
	}
	if _, err := os.Stat(from); errors.Is(err, fs.ErrNotExist) {
		return false, false, nil
	} else if err != nil {
		return false, false, fmt.Errorf("stat %s: %w", from, err)
	}
	if _, err := os.Stat(to); err == nil {
		return false, true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, false, fmt.Errorf("stat %s: %w", to, err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return false, false, fmt.Errorf("create %s: %w", filepath.Dir(to), err)
	}
	if err := os.Rename(from, to); err == nil {
		return true, false, nil
	}

	data, err := os.ReadFile(from)
	if err != nil {
		return false, false, fmt.Errorf("read %s: %w", from, err)
	}
	if err := writeAtomic(to, data); err != nil {
		return false, false, err
	}
	if err := os.Remove(from); err != nil {
		return false, false, fmt.Errorf("remove %s: %w", from, err)
	}
	return true, false, nil
}

// oldSiblingPath mirrors the pre-directory naming convention:
// <state-without-extension>-buckets.json.
func oldSiblingPath(statePath, suffix string) string {
	ext := filepath.Ext(statePath)
	if ext != "" {
		return strings.TrimSuffix(statePath, ext) + suffix + ext
	}
	return statePath + suffix + ".json"
}
