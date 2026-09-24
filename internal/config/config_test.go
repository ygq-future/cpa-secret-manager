package config

import (
	"strings"
	"testing"
)

func TestLoadBytes_ExtractsStatePathFromHostConfig(t *testing.T) {
	hostConfig := strings.Join([]string{
		"plugins:",
		"  enabled: true",
		"  dir: \"plugins\"",
		"  configs:",
		"    other-plugin:",
		"      state_cache_path: \"data/other.json\"",
		"    cpa-secret-manager:",
		"      enabled: true",
		"      priority: 1",
		"      state_path: \"data/custom-secret-manager.json\"",
		"request-retry: 3",
		"",
	}, "\n")

	cfg, warnings, err := LoadBytes([]byte(hostConfig))
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("LoadBytes() warnings = %v, want none", warnings)
	}
	if cfg.StatePath != "data/custom-secret-manager.json" {
		t.Fatalf("StatePath = %q, want data/custom-secret-manager.json", cfg.StatePath)
	}
}

func TestLoadBytes_AcceptsBarePluginSection(t *testing.T) {
	cfg, _, err := LoadBytes([]byte("enabled: true\nstate_path: \"data/bare.json\"\n"))
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	if cfg.StatePath != "data/bare.json" {
		t.Fatalf("StatePath = %q, want data/bare.json", cfg.StatePath)
	}
}

func TestLoadBytes_FallsBackWhenSectionMissing(t *testing.T) {
	hostConfig := "plugins:\n  enabled: true\n  configs:\n    other-plugin:\n      enabled: true\n"

	cfg, warnings, err := LoadBytes([]byte(hostConfig))
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	if cfg.StatePath != DefaultStatePath {
		t.Fatalf("StatePath = %q, want default %q", cfg.StatePath, DefaultStatePath)
	}
	if len(warnings) != 0 {
		t.Fatalf("LoadBytes() warnings = %v, want none for missing section", warnings)
	}
}

func TestLoadBytes_RejectsBlankStatePath(t *testing.T) {
	cfg, warnings, err := LoadBytes([]byte("state_path: \"\"\n"))
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	if cfg.StatePath != DefaultStatePath {
		t.Fatalf("StatePath = %q, want default %q", cfg.StatePath, DefaultStatePath)
	}
	if len(warnings) == 0 {
		t.Fatal("LoadBytes() returned no warning for a blank state path")
	}
}

func TestLoadBytes_AcceptsJSONSection(t *testing.T) {
	cfg, _, err := LoadBytes([]byte(`{"state_path":"data/json.json"}`))
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	if cfg.StatePath != "data/json.json" {
		t.Fatalf("StatePath = %q, want data/json.json", cfg.StatePath)
	}
}

func TestLoadBytes_EmptyInputUsesDefaults(t *testing.T) {
	cfg, _, err := LoadBytes(nil)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	if cfg != Default() {
		t.Fatalf("LoadBytes(nil) = %+v, want %+v", cfg, Default())
	}
}
