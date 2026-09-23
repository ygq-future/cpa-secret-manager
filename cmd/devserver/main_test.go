package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"cpa-secret-manager/internal/keys"
	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/remarks"
)

const testManagementKey = "test-management-key"

func newTestHost(t *testing.T) (*devHost, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	host, err := newDevHost(devHostOptions{
		ManagementKey: testManagementKey,
		KeysPath:      filepath.Join(dir, "api-keys.json"),
		StatePath:     filepath.Join(dir, "state.json"),
	})
	if err != nil {
		t.Fatalf("newDevHost() error = %v", err)
	}
	server := httptest.NewServer(host.handler())
	t.Cleanup(func() {
		server.Close()
		if err := host.shutdown(context.Background()); err != nil {
			t.Errorf("shutdown() error = %v", err)
		}
	})
	return host, server
}

func call(t *testing.T, server *httptest.Server, method string, path string, body string, authorized bool) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if authorized {
		request.Header.Set("Authorization", "Bearer "+testManagementKey)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return response.StatusCode, payload
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return out
}

func TestDevHost_RequiresManagementKey(t *testing.T) {
	_, server := newTestHost(t)

	if status, _ := call(t, server, http.MethodGet, "/v0/management/api-keys", "", false); status != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", status)
	}
	if status, _ := call(t, server, http.MethodGet, "/v0/management/plugins/cpa-secret-manager/settings", "", false); status != http.StatusUnauthorized {
		t.Fatalf("unauthorized plugin route status = %d, want 401", status)
	}
	if status, _ := call(t, server, http.MethodGet, "/v0/resource/plugins/cpa-secret-manager/keys", "", false); status != http.StatusOK {
		t.Fatalf("resource status = %d, want 200 without management auth", status)
	}
}

// TestDevHost_KeyLifecycleEndToEnd walks the full operator flow: seed keys,
// inspect resolved metadata, edit the key list, generate a key and confirm the
// remark lifecycle.
func TestDevHost_KeyLifecycleEndToEnd(t *testing.T) {
	_, server := newTestHost(t)

	if status, body := call(t, server, http.MethodPost, "/dev/seed?keys=2", "", true); status != http.StatusOK {
		t.Fatalf("seed status = %d: %s", status, body)
	}

	status, body := call(t, server, http.MethodGet, "/v0/management/api-keys", "", true)
	if status != http.StatusOK {
		t.Fatalf("list status = %d: %s", status, body)
	}
	list := decode[struct {
		APIKeys []string `json:"api-keys"`
	}](t, body)
	if len(list.APIKeys) != 2 {
		t.Fatalf("seeded keys = %d, want 2", len(list.APIKeys))
	}

	status, body = call(t, server, http.MethodPost, "/v0/management/plugins/cpa-secret-manager/resolve",
		`{"keys":`+mustJSON(t, list.APIKeys)+`}`, true)
	if status != http.StatusOK {
		t.Fatalf("resolve status = %d: %s", status, body)
	}
	resolved := decode[management.ResolveResult](t, body)
	if len(resolved.Items) != 2 {
		t.Fatalf("resolved items = %d, want 2", len(resolved.Items))
	}
	for index, item := range resolved.Items {
		if item.Remark == "" {
			t.Fatalf("items[%d] has no seeded remark", index)
		}
		if item.Usage == nil || item.Usage.Requests != 3 || len(item.Usage.Models) != 3 {
			t.Fatalf("items[%d] usage = %+v, want 3 requests across 3 models", index, item.Usage)
		}
	}
	if resolved.UnattributedRequests != 1 {
		t.Fatalf("unattributed = %d, want the seeded unattributed record", resolved.UnattributedRequests)
	}

	// Rename the first key and move its remark, like the page does.
	replacement := "sk-renamed-key"
	updated := append([]string{replacement}, list.APIKeys[1:]...)
	if status, body := call(t, server, http.MethodPut, "/v0/management/api-keys", mustJSON(t, updated), true); status != http.StatusOK {
		t.Fatalf("replace status = %d: %s", status, body)
	}
	if status, body := call(t, server, http.MethodPut, "/v0/management/plugins/cpa-secret-manager/remarks",
		`{"key":"`+replacement+`","remark":"renamed"}`, true); status != http.StatusOK {
		t.Fatalf("remark status = %d: %s", status, body)
	}
	if status, body := call(t, server, http.MethodPut, "/v0/management/plugins/cpa-secret-manager/remarks",
		`{"key":"`+list.APIKeys[0]+`","remark":""}`, true); status != http.StatusOK {
		t.Fatalf("clear status = %d: %s", status, body)
	}

	status, body = call(t, server, http.MethodPost, "/v0/management/plugins/cpa-secret-manager/resolve",
		`{"keys":`+mustJSON(t, updated)+`}`, true)
	if status != http.StatusOK {
		t.Fatalf("resolve after rename status = %d: %s", status, body)
	}
	resolved = decode[management.ResolveResult](t, body)
	if resolved.Items[0].Remark != "renamed" {
		t.Fatalf("renamed remark = %q, want renamed", resolved.Items[0].Remark)
	}
	if resolved.Items[0].Usage != nil {
		t.Fatalf("renamed key usage = %+v, want no usage (attribution is per key material)", resolved.Items[0].Usage)
	}
	if resolved.OrphanUsage != 1 {
		t.Fatalf("orphan_usage = %d, want 1 for the removed key", resolved.OrphanUsage)
	}

	// The generated key must match the official algorithm shape.
	status, body = call(t, server, http.MethodPost, "/v0/management/plugins/cpa-secret-manager/keys/generate", "", true)
	if status != http.StatusOK {
		t.Fatalf("generate status = %d: %s", status, body)
	}
	generated := decode[management.GenerateResult](t, body)
	if len(generated.Key) != len(keys.Prefix)+keys.RandomLength || !strings.HasPrefix(generated.Key, keys.Prefix) {
		t.Fatalf("generated key = %q, want the official shape", generated.Key)
	}
	if suffix := strings.TrimPrefix(generated.Key, keys.Prefix); strings.Trim(suffix, keys.Charset) != "" {
		t.Fatalf("generated key %q escapes the official charset", generated.Key)
	}

	// Deleting the last key removes it from the store.
	if status, _ := call(t, server, http.MethodDelete, "/v0/management/api-keys?value="+updated[1], "", true); status != http.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	_, body = call(t, server, http.MethodGet, "/v0/management/api-keys", "", true)
	list = decode[struct {
		APIKeys []string `json:"api-keys"`
	}](t, body)
	if len(list.APIKeys) != 1 {
		t.Fatalf("keys after delete = %d, want 1", len(list.APIKeys))
	}
}

func TestDevHost_ServesPagesAndPluginList(t *testing.T) {
	_, server := newTestHost(t)

	status, body := call(t, server, http.MethodGet, "/control-panel", "", false)
	if status != http.StatusOK || !strings.Contains(string(body), "plugin-frame") {
		t.Fatalf("control panel status = %d, want the iframe harness", status)
	}

	status, body = call(t, server, http.MethodGet, "/v0/resource/plugins/cpa-secret-manager/keys", "", false)
	if status != http.StatusOK {
		t.Fatalf("page status = %d", status)
	}
	if !strings.Contains(string(body), "v"+mustVersion(t)) {
		t.Fatalf("plugin page is missing the version badge")
	}

	status, body = call(t, server, http.MethodGet, "/v0/management/plugins", "", true)
	if status != http.StatusOK {
		t.Fatalf("plugin list status = %d", status)
	}
	list := decode[struct {
		PluginsEnabled bool `json:"pluginsEnabled"`
		Plugins        []struct {
			ID               string `json:"id"`
			EffectiveEnabled bool   `json:"effectiveEnabled"`
			Menus            []struct {
				Path string `json:"path"`
				Menu string `json:"menu"`
			} `json:"menus"`
			Metadata struct {
				Version string `json:"version"`
			} `json:"metadata"`
		} `json:"plugins"`
	}](t, body)
	if !list.PluginsEnabled || len(list.Plugins) != 1 {
		t.Fatalf("plugin list = %+v, want one effective plugin", list)
	}
	if list.Plugins[0].ID != management.PluginID || !list.Plugins[0].EffectiveEnabled {
		t.Fatalf("plugin entry = %+v, want the registered plugin", list.Plugins[0])
	}
	if len(list.Plugins[0].Menus) != 1 || list.Plugins[0].Menus[0].Path != "/v0/resource/plugins/"+management.PluginID+management.PathPage {
		t.Fatalf("plugin menus = %+v, want the resource page entry", list.Plugins[0].Menus)
	}
	if list.Plugins[0].Menus[0].Menu != management.MenuLabel {
		t.Fatalf("menu label = %q, want %q", list.Plugins[0].Menus[0].Menu, management.MenuLabel)
	}
}

func TestDevHost_UsageInjectionAttributesByKey(t *testing.T) {
	_, server := newTestHost(t)

	if status, _ := call(t, server, http.MethodPut, "/v0/management/api-keys", `["sk-manual"]`, true); status != http.StatusOK {
		t.Fatal("failed to prepare the key list")
	}
	if status, body := call(t, server, http.MethodPost,
		"/dev/simulate-usage?key=sk-manual&model=gpt-5.6&count=2&input=10&output=20&total=30", "", true); status != http.StatusOK {
		t.Fatalf("simulate status = %d: %s", status, body)
	}

	status, body := call(t, server, http.MethodPost, "/v0/management/plugins/cpa-secret-manager/resolve", `{"keys":["sk-manual"]}`, true)
	if status != http.StatusOK {
		t.Fatalf("resolve status = %d", status)
	}
	resolved := decode[management.ResolveResult](t, body)
	usage := resolved.Items[0].Usage
	if usage == nil || usage.Requests != 2 || usage.Input != 20 || usage.Output != 40 || usage.Total != 60 {
		t.Fatalf("usage = %+v, want 2 requests and doubled counters", usage)
	}
	if usage.Models[0].Model != "gpt-5.6" {
		t.Fatalf("models = %+v, want gpt-5.6", usage.Models)
	}
	if resolved.Items[0].Hash != remarks.Index("sk-manual") {
		t.Fatalf("hash = %q, want the digest of the seeded key", resolved.Items[0].Hash)
	}
}

func TestDevHost_RejectsMalformedRequests(t *testing.T) {
	_, server := newTestHost(t)

	if status, _ := call(t, server, http.MethodPut, "/v0/management/plugins/cpa-secret-manager/remarks", `{"key":"","remark":"x"}`, true); status != http.StatusBadRequest {
		t.Fatalf("blank key status = %d, want 400", status)
	}
	if status, _ := call(t, server, http.MethodPost, "/dev/simulate-usage?count=5000", "", true); status != http.StatusBadRequest {
		t.Fatalf("oversized count status = %d, want 400", status)
	}
	if status, _ := call(t, server, http.MethodPut, "/v0/management/api-keys", `{not json`, true); status != http.StatusBadRequest {
		t.Fatalf("malformed key list status = %d, want 400", status)
	}
}

// TestDevHost_KeyStoreSurvivesRestart guards the simulator against a store
// format it cannot read back, which would silently drop every seeded key.
func TestDevHost_KeyStoreSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	keysPath := filepath.Join(dir, "api-keys.json")
	statePath := filepath.Join(dir, "state.json")

	first, err := newDevHost(devHostOptions{ManagementKey: testManagementKey, KeysPath: keysPath, StatePath: statePath})
	if err != nil {
		t.Fatalf("newDevHost() error = %v", err)
	}
	server := httptest.NewServer(first.handler())
	if status, body := call(t, server, http.MethodPut, "/v0/management/api-keys", `["sk-alpha","sk-beta"]`, true); status != http.StatusOK {
		t.Fatalf("replace status = %d: %s", status, body)
	}
	server.Close()
	if err := first.shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}

	restarted, err := newDevHost(devHostOptions{ManagementKey: testManagementKey, KeysPath: keysPath, StatePath: statePath})
	if err != nil {
		t.Fatalf("restart newDevHost() error = %v", err)
	}
	defer func() {
		if err := restarted.shutdown(context.Background()); err != nil {
			t.Errorf("shutdown() error = %v", err)
		}
	}()

	loaded := restarted.keyList()
	if len(loaded) != 2 || loaded[0] != "sk-alpha" || loaded[1] != "sk-beta" {
		t.Fatalf("reloaded keys = %v, want the persisted list", loaded)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(raw)
}

func mustVersion(t *testing.T) string {
	t.Helper()
	version := ""
	if index := strings.Index(management.PageHTML, "version-badge\">v"); index >= 0 {
		rest := management.PageHTML[index+len("version-badge\">v"):]
		if end := strings.Index(rest, "<"); end >= 0 {
			version = rest[:end]
		}
	}
	if version == "" {
		t.Fatal("could not read the version badge from the page")
	}
	return version
}
