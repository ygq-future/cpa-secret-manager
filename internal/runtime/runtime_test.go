package runtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cpa-secret-manager/internal/keys"
	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/remarks"
	"cpa-secret-manager/internal/state"
	"cpa-secret-manager/internal/version"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, raw []byte) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode envelope %s: %v", raw, err)
	}
	return env
}

func registerRuntime(t *testing.T, statePath string) *Runtime {
	t.Helper()
	return registerRuntimeWith(t, Options{StatePath: statePath})
}

func registerRuntimeWith(t *testing.T, opts Options) *Runtime {
	t.Helper()
	rt := New(opts)
	request := fmt.Sprintf(`{"config_yaml":%q}`, "state_path: "+filepath.ToSlash(opts.StatePath)+"\n")
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodPluginRegister, []byte(request)))
	if !env.OK {
		t.Fatalf("register failed: %s", env.Result)
	}
	return rt
}

func usageCall(t *testing.T, rt *Runtime, payload map[string]any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode usage record: %v", err)
	}
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodUsageHandle, raw))
	if !env.OK {
		t.Fatalf("usage.handle failed: %+v", env.Error)
	}
}

func resolveResult(t *testing.T, rt *Runtime, apiKeys []string) management.ResolveResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{"keys": apiKeys})
	if err != nil {
		t.Fatalf("encode resolve: %v", err)
	}
	status, responseBody := managementCall(t, rt, "POST", management.RouteResolve, body)
	if status != 200 {
		t.Fatalf("resolve status = %d, want 200", status)
	}
	var result management.ResolveResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		t.Fatalf("decode resolve: %v", err)
	}
	if len(result.Items) != len(apiKeys) {
		t.Fatalf("items = %d, want one per submitted key", len(result.Items))
	}
	return result
}

func managementCall(t *testing.T, rt *Runtime, method string, path string, body []byte) (int, []byte) {
	t.Helper()
	payload := map[string]any{"Method": method, "Path": path}
	if body != nil {
		payload["Body"] = base64.StdEncoding.EncodeToString(body)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode management request: %v", err)
	}

	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodManagementHandle, raw))
	if !env.OK {
		t.Fatalf("management.handle failed: %+v", env.Error)
	}
	var response struct {
		StatusCode int    `json:"StatusCode"`
		Body       []byte `json:"Body"`
	}
	if err := json.Unmarshal(env.Result, &response); err != nil {
		t.Fatalf("decode management response: %v", err)
	}
	return response.StatusCode, response.Body
}

func TestRegister_ReportsMetadataAndCapabilities(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodPluginRegister, []byte(`{"config_yaml":""}`)))
	var result RegisterResult
	if err := json.Unmarshal(env.Result, &result); err != nil {
		t.Fatalf("decode register result: %v", err)
	}
	if result.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %d, want %d", result.SchemaVersion, SchemaVersion)
	}
	if result.Metadata.Version != version.PluginVersion {
		t.Fatalf("metadata version = %q, want %q", result.Metadata.Version, version.PluginVersion)
	}
	if result.Metadata.Name == "" || result.Metadata.GitHubRepository == "" {
		t.Fatalf("metadata = %+v, want name and repository", result.Metadata)
	}
	if len(result.Metadata.ConfigFields) != 0 {
		t.Fatalf("ConfigFields = %+v, want none (business settings live in the state document)", result.Metadata.ConfigFields)
	}
	if !result.Capabilities.ManagementAPI {
		t.Fatal("management_api capability = false, want true")
	}
}

func TestManagementRegistration_DeclaresSettingsRoutes(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodManagementRegister, nil))
	var registration managementRegistration
	if err := json.Unmarshal(env.Result, &registration); err != nil {
		t.Fatalf("decode management registration: %v", err)
	}

	declared := map[string]bool{}
	for _, route := range registration.Routes {
		declared[route.Method+" "+route.Path] = true
	}
	if !declared["GET "+management.RouteSettings] || !declared["PUT "+management.RouteSettings] {
		t.Fatalf("routes = %+v, want GET and PUT for %s", registration.Routes, management.RouteSettings)
	}
}

func TestSettings_UpdateIsPersistedAndReloadable(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "nested", "state.json")
	rt := registerRuntime(t, statePath)

	status, body := managementCall(t, rt, "GET", "/v0/management/plugins/"+management.PluginID+"/settings", nil)
	if status != 200 {
		t.Fatalf("GET settings status = %d, want 200", status)
	}
	var settings management.Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if !settings.UsageEnabled {
		t.Fatal("usage_enabled = false, want the default true")
	}
	if filepath.ToSlash(settings.StatePath) != filepath.ToSlash(statePath) {
		t.Fatalf("state_path = %q, want %q", settings.StatePath, statePath)
	}

	status, _ = managementCall(t, rt, "PUT", "/v0/management/plugins/"+management.PluginID+"/settings",
		[]byte(`{"usage_enabled":false}`))
	if status != 200 {
		t.Fatalf("PUT settings status = %d, want 200", status)
	}

	reloaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if reloaded.AppConfig().UsageEnabled {
		t.Fatal("persisted usage_enabled = true, want false")
	}
}

func TestSettings_SurfaceCorruptStateWarning(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(statePath, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	rt := registerRuntime(t, statePath)

	_, body := managementCall(t, rt, "GET", management.PathSettings, nil)
	var settings management.Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings.StateWarning == "" {
		t.Fatal("state_warning is empty for a corrupt state document")
	}
}

func TestReconfigure_SwitchesStateFile(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first.json")
	second := filepath.Join(t.TempDir(), "second.json")
	rt := registerRuntime(t, first)

	request := fmt.Sprintf(`{"config_yaml":%q}`, "state_path: "+filepath.ToSlash(second)+"\n")
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodPluginReconfigure, []byte(request)))
	if !env.OK {
		t.Fatalf("reconfigure failed: %+v", env.Error)
	}

	_, body := managementCall(t, rt, "GET", management.PathSettings, nil)
	var settings management.Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if filepath.ToSlash(settings.StatePath) != filepath.ToSlash(second) {
		t.Fatalf("state_path = %q, want %q", settings.StatePath, second)
	}
}

func TestShutdown_PersistsOnceAndRejectsFurtherWrites(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file missing after shutdown: %v", err)
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if err := rt.UpdateSettings(context.Background(), false); err != ErrShutdown {
		t.Fatalf("UpdateSettings() error = %v, want ErrShutdown", err)
	}
}

func TestResolve_AlignsItemsWithSubmittedOrder(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	remarkBody, err := json.Marshal(map[string]string{"key": "sk-two", "remark": "  second key  "})
	if err != nil {
		t.Fatalf("encode remark: %v", err)
	}
	if status, _ := managementCall(t, rt, "PUT", management.RouteRemarks, remarkBody); status != 200 {
		t.Fatalf("remark status = %d, want 200", status)
	}

	resolveBody, err := json.Marshal(map[string]any{"keys": []string{"sk-one", "sk-two", "sk-three"}})
	if err != nil {
		t.Fatalf("encode resolve: %v", err)
	}
	status, body := managementCall(t, rt, "POST", management.RouteResolve, resolveBody)
	if status != 200 {
		t.Fatalf("resolve status = %d, want 200", status)
	}

	var result management.ResolveResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode resolve: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("items = %d, want one per submitted key", len(result.Items))
	}
	if result.Items[0].Hash != remarks.Index("sk-one") {
		t.Fatalf("items[0].hash = %q, want the digest of sk-one", result.Items[0].Hash)
	}
	if result.Items[0].Remark != "" || result.Items[2].Remark != "" {
		t.Fatalf("unexpected remark on an unannotated key: %+v", result.Items)
	}
	if result.Items[1].Remark != "second key" {
		t.Fatalf("items[1].remark = %q, want the trimmed remark", result.Items[1].Remark)
	}
	if result.OrphanRemarks != 0 {
		t.Fatalf("orphan_remarks = %d, want 0 while every stored remark matches a submitted key", result.OrphanRemarks)
	}
}

func TestSetRemark_PersistsAndClears(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	body, err := json.Marshal(map[string]string{"key": "sk-dev", "remark": "local development"})
	if err != nil {
		t.Fatalf("encode remark: %v", err)
	}
	if status, _ := managementCall(t, rt, "PUT", management.RouteRemarks, body); status != 200 {
		t.Fatalf("remark status = %d, want 200", status)
	}

	reloaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("reload state: %v", err)
	}
	entry, ok := reloaded.Remark(remarks.Index("sk-dev"))
	if !ok || entry.Remark != "local development" {
		t.Fatalf("persisted remark = %+v (ok=%v), want local development", entry, ok)
	}
	if entry.UpdatedAt.IsZero() {
		t.Fatal("persisted remark has no updated_at timestamp")
	}

	clearBody, err := json.Marshal(map[string]string{"key": "sk-dev", "remark": ""})
	if err != nil {
		t.Fatalf("encode clear: %v", err)
	}
	if status, _ := managementCall(t, rt, "PUT", management.RouteRemarks, clearBody); status != 200 {
		t.Fatalf("clear status = %d, want 200", status)
	}
	if _, ok := rt.Store().Remark(remarks.Index("sk-dev")); ok {
		t.Fatal("remark still present after being cleared")
	}
}

func TestSetRemark_RejectsMissingKeyAndOverlongText(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	for name, payload := range map[string]map[string]string{
		"blank key":     {"key": "   ", "remark": "x"},
		"overlong text": {"key": "sk-a", "remark": strings.Repeat("备", remarks.MaxLength+1)},
	} {
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode %s: %v", name, err)
		}
		if status, _ := managementCall(t, rt, "PUT", management.RouteRemarks, body); status != 400 {
			t.Fatalf("%s status = %d, want 400", name, status)
		}
	}
}

func TestGenerateKey_MatchesOfficialAlgorithm(t *testing.T) {
	source := make([]byte, 0, 128)
	for value := keys.MaxUnbiasedByte; value <= 255; value++ {
		source = append(source, byte(value))
	}
	for index := range keys.RandomLength {
		source = append(source, byte(index))
	}
	source = append(source, make([]byte, 128-len(source))...)

	rt := New(Options{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
		Random:    bytes.NewReader(source),
	})
	status, body := managementCall(t, rt, "POST", management.RouteGenerate, nil)
	if status != 200 {
		t.Fatalf("generate status = %d, want 200", status)
	}
	var result management.GenerateResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode generate result: %v", err)
	}
	if want := keys.Prefix + keys.Charset[:keys.RandomLength]; result.Key != want {
		t.Fatalf("generated key = %q, want %q", result.Key, want)
	}
}

func TestUsage_AttributesTokensPerKeyAndModel(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	for _, payload := range []map[string]any{
		{"APIKey": "sk-a", "Model": "gpt-5.6", "Detail": map[string]any{"InputTokens": 10, "OutputTokens": 20, "TotalTokens": 30}},
		{"APIKey": "sk-a", "Model": "gpt-5.6", "Failed": true, "Detail": map[string]any{"InputTokens": 1, "TotalTokens": 1}},
		{"APIKey": "sk-a", "Model": "claude-sonnet", "Detail": map[string]any{"InputTokens": 5, "CacheReadTokens": 7, "TotalTokens": 12}},
		{"APIKey": "sk-b", "Model": "gpt-5.6", "Detail": map[string]any{"TotalTokens": 100}},
	} {
		usageCall(t, rt, payload)
	}

	result := resolveResult(t, rt, []string{"sk-a", "sk-b"})
	first := result.Items[0].Usage
	if first == nil {
		t.Fatal("no usage recorded for sk-a")
	}
	if first.Requests != 3 || first.Failed != 1 {
		t.Fatalf("requests = %d, failed = %d; want 3 and 1", first.Requests, first.Failed)
	}
	if first.Input != 16 || first.Output != 20 || first.CacheRead != 7 || first.Total != 43 {
		t.Fatalf("counters = %+v, want input 16, output 20, cache read 7, total 43", first.Counters)
	}
	if first.FirstSeen == "" || first.LastSeen == "" {
		t.Fatalf("usage window = %q..%q, want timestamps", first.FirstSeen, first.LastSeen)
	}
	if len(first.Models) != 2 {
		t.Fatalf("models = %+v, want two model rows", first.Models)
	}
	// Ordered by token volume so the page renders a stable list.
	if first.Models[0].Model != "gpt-5.6" || first.Models[0].Total != 31 || first.Models[0].Requests != 2 {
		t.Fatalf("models[0] = %+v, want gpt-5.6 with 2 requests and 31 tokens", first.Models[0])
	}
	if first.Models[1].Model != "claude-sonnet" {
		t.Fatalf("models[1] = %+v, want claude-sonnet", first.Models[1])
	}

	if result.Items[1].Usage == nil || result.Items[1].Usage.Total != 100 {
		t.Fatalf("sk-b usage = %+v, want 100 total tokens", result.Items[1].Usage)
	}
	if result.Items[0].Hash != remarks.Index("sk-a") {
		t.Fatalf("items[0].hash = %q, want the digest of sk-a", result.Items[0].Hash)
	}
}

func TestUsage_KeepsUnattributedAndOrphanCounts(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	usageCall(t, rt, map[string]any{"Model": "gpt-5.6", "Detail": map[string]any{"TotalTokens": 5}})
	usageCall(t, rt, map[string]any{"APIKey": "sk-retired", "Model": "gpt-5.6", "Detail": map[string]any{"TotalTokens": 6}})

	result := resolveResult(t, rt, []string{"sk-a"})
	if result.UnattributedRequests != 1 {
		t.Fatalf("unattributed_requests = %d, want 1", result.UnattributedRequests)
	}
	if result.OrphanUsage != 1 {
		t.Fatalf("orphan_usage = %d, want 1 for the key that is no longer listed", result.OrphanUsage)
	}
	if result.Items[0].Usage != nil {
		t.Fatalf("items[0].usage = %+v, want no usage for an unused key", result.Items[0].Usage)
	}
}

func TestUsage_DisabledSettingStopsCollection(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	if status, _ := managementCall(t, rt, "PUT", management.RouteSettings, []byte(`{"usage_enabled":false}`)); status != 200 {
		t.Fatalf("settings status = %d, want 200", status)
	}
	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "Detail": map[string]any{"TotalTokens": 9}})
	usageCall(t, rt, map[string]any{"Model": "m", "Detail": map[string]any{"TotalTokens": 9}})

	result := resolveResult(t, rt, []string{"sk-a"})
	if result.Items[0].Usage != nil {
		t.Fatalf("usage = %+v, want nothing recorded while collection is disabled", result.Items[0].Usage)
	}
	if result.UnattributedRequests != 0 {
		t.Fatalf("unattributed_requests = %d, want 0 while collection is disabled", result.UnattributedRequests)
	}
}

func TestUsageHandle_AcceptsSnakeCasePayload(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	raw := []byte(`{"api_key":"sk-a","model":"gpt-5.6","failed":true,"requested_at":"2026-09-23T10:00:00Z",` +
		`"detail":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}`)
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodUsageHandle, raw))
	if !env.OK {
		t.Fatalf("usage.handle failed: %+v", env.Error)
	}

	result := resolveResult(t, rt, []string{"sk-a"})
	usageRecord := result.Items[0].Usage
	if usageRecord == nil || usageRecord.Input != 3 || usageRecord.Output != 4 || usageRecord.Total != 7 || usageRecord.Failed != 1 {
		t.Fatalf("usage = %+v, want the snake_case payload counters", usageRecord)
	}
	if usageRecord.FirstSeen != "2026-09-23T10:00:00Z" {
		t.Fatalf("first_seen = %q, want the payload timestamp", usageRecord.FirstSeen)
	}
}

func TestFlushUsage_PersistsAggregateInBackground(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntimeWith(t, Options{StatePath: statePath, FlushInterval: 10 * time.Millisecond})
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "Detail": map[string]any{"TotalTokens": 42}})

	deadline := time.Now().Add(2 * time.Second)
	for {
		reloaded, err := state.Load(statePath)
		if err == nil {
			if entry, ok := reloaded.Usage()[remarks.Index("sk-a")]; ok && entry.Total == 42 {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("usage counters were not persisted by the background flush loop")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestShutdown_PersistsPendingUsageAndSeedsOnRestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntimeWith(t, Options{StatePath: statePath, FlushInterval: time.Hour})

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "Detail": map[string]any{"TotalTokens": 42}})
	usageCall(t, rt, map[string]any{"Model": "m", "Detail": map[string]any{"TotalTokens": 1}})
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	restarted := registerRuntime(t, statePath)
	result := resolveResult(t, restarted, []string{"sk-a"})
	if result.Items[0].Usage == nil || result.Items[0].Usage.Total != 42 {
		t.Fatalf("usage after restart = %+v, want the persisted 42 tokens", result.Items[0].Usage)
	}
	if result.UnattributedRequests != 1 {
		t.Fatalf("unattributed_requests = %d, want the persisted 1", result.UnattributedRequests)
	}
}

func TestHandle_UnknownMethodFails(t *testing.T) {
	rt := New(Options{StatePath: filepath.Join(t.TempDir(), "state.json")})

	env := decodeEnvelope(t, rt.Handle(context.Background(), "plugin.bogus", nil))
	if env.OK {
		t.Fatal("unknown method returned ok = true")
	}
	if env.Error == nil || env.Error.Code != "unknown_method" {
		t.Fatalf("error = %+v, want code unknown_method", env.Error)
	}
}

func TestHandle_InvalidConfigFails(t *testing.T) {
	rt := New(Options{StatePath: filepath.Join(t.TempDir(), "state.json")})

	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodPluginRegister, []byte(`{"config_yaml":"{invalid json"}`)))
	if env.OK {
		t.Fatal("invalid config returned ok = true")
	}
	if env.Error == nil || env.Error.Code != "invalid_config" {
		t.Fatalf("error = %+v, want code invalid_config", env.Error)
	}
}
