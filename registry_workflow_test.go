package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"cpa-secret-manager/internal/management"
	pluginruntime "cpa-secret-manager/internal/runtime"
)

type registryDocument struct {
	SchemaVersion int `json:"schema_version"`
	Plugins       []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Author      string   `json:"author"`
		Version     string   `json:"version"`
		Repository  string   `json:"repository"`
		Homepage    string   `json:"homepage"`
		License     string   `json:"license"`
		Tags        []string `json:"tags"`
	} `json:"plugins"`
}

func readRegistry(t *testing.T) registryDocument {
	t.Helper()
	raw, err := os.ReadFile("registry.json")
	if err != nil {
		t.Fatalf("read registry.json: %v", err)
	}
	var doc registryDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode registry.json: %v", err)
	}
	return doc
}

// TestRegistry_ContractMatchesPluginStoreSchema guards the CPA plugin store
// requirements: schema version, mandatory fields and a GitHub repository URL.
func TestRegistry_ContractMatchesPluginStoreSchema(t *testing.T) {
	doc := readRegistry(t)
	if doc.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", doc.SchemaVersion)
	}
	if len(doc.Plugins) != 1 {
		t.Fatalf("plugins = %d entries, want exactly 1", len(doc.Plugins))
	}

	entry := doc.Plugins[0]
	for name, value := range map[string]string{
		"id":          entry.ID,
		"name":        entry.Name,
		"description": entry.Description,
		"author":      entry.Author,
		"repository":  entry.Repository,
	} {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("registry field %s is empty", name)
		}
	}
	if !regexp.MustCompile(`^https://github\.com/[^/]+/[^/]+$`).MatchString(entry.Repository) {
		t.Fatalf("repository = %q, want https://github.com/{owner}/{repo}", entry.Repository)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(entry.Version) {
		t.Fatalf("version = %q, want a three-part semantic version without a v prefix", entry.Version)
	}
	if len(entry.Tags) == 0 {
		t.Fatal("registry tags are empty")
	}
}

// TestVersionConsistency keeps the registry entry and the runtime metadata in
// lockstep. The embedded page badge joins this gate once the page exists.
func TestVersionConsistency(t *testing.T) {
	doc := readRegistry(t)
	if got := doc.Plugins[0].Version; got != pluginruntime.PluginVersion {
		t.Fatalf("registry version = %q, runtime version = %q; they must match", got, pluginruntime.PluginVersion)
	}
}

func TestRegistry_IDMatchesPluginID(t *testing.T) {
	doc := readRegistry(t)
	if got := doc.Plugins[0].ID; got != management.PluginID {
		t.Fatalf("registry id = %q, management plugin id = %q; they must match", got, management.PluginID)
	}
}

// TestReleaseWorkflow_BuildsTheRegisteredPluginID guards the release pipeline
// against a plugin name mismatch, which would ship a dynamic library the host
// cannot map to plugins.configs.<pluginID>.
func TestReleaseWorkflow_BuildsTheRegisteredPluginID(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	want := "PLUGIN_NAME: " + management.PluginID
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release workflow does not declare %q", want)
	}
}

func TestDomainDocumentation_IsPresent(t *testing.T) {
	for _, path := range []string{
		"CONTEXT.md",
		"AGENTS.md",
		"CLAUDE.md",
		"docs/requirements/v1.0.0-roadmap.md",
		"docs/agents/domain.md",
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("required document %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("required document %s is empty", path)
		}
	}

	adrs, err := filepath.Glob(filepath.Join("docs", "adr", "*.md"))
	if err != nil {
		t.Fatalf("glob ADRs: %v", err)
	}
	if len(adrs) == 0 {
		t.Fatal("no architecture decision records found under docs/adr")
	}
}
