package runtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cpa-secret-manager/internal/keys"
	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/remarks"
	"cpa-secret-manager/internal/state"
	"cpa-secret-manager/internal/usage"
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

func resolveWindow(t *testing.T, rt *Runtime, apiKeys []string, start, end time.Time) management.ResolveResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"keys":  apiKeys,
		"start": start.UTC().Format(time.RFC3339),
		"end":   end.UTC().Format(time.RFC3339),
	})
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
	if !result.Capabilities.RequestInterceptor {
		t.Fatal("request_interceptor capability = false, want true")
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
	disabled := false
	if err := rt.UpdateSettings(context.Background(), management.SettingsUpdate{UsageEnabled: &disabled}); err != ErrShutdown {
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

func TestResolve_NeverDeletesStoredMetadata(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	remarkBody, err := json.Marshal(map[string]string{"key": "sk-retired", "remark": "keep me"})
	if err != nil {
		t.Fatalf("encode remark: %v", err)
	}
	if status, _ := managementCall(t, rt, "PUT", management.RouteRemarks, remarkBody); status != 200 {
		t.Fatalf("remark status = %d, want 200", status)
	}
	usageCall(t, rt, map[string]any{"APIKey": "sk-retired", "Model": "gpt-5.6", "Detail": map[string]any{"TotalTokens": 6}})

	// A key list that omits the stored key must not touch it: reading is read-only,
	// and the omission is only reported back as a stale hash.
	omitted := resolveResult(t, rt, []string{"sk-a"})
	if omitted.Items[0].Usage != nil {
		t.Fatalf("items[0].usage = %+v, want no usage for an unused key", omitted.Items[0].Usage)
	}
	if len(omitted.StaleHashes) != 1 || omitted.StaleHashes[0] != remarks.Index("sk-retired") {
		t.Fatalf("stale_hashes = %v, want the digest of the unlisted key", omitted.StaleHashes)
	}
	retained := resolveResult(t, rt, []string{"sk-retired"})
	if retained.Items[0].Remark != "keep me" {
		t.Fatalf("remark = %q, want the stored remark to survive a resolve", retained.Items[0].Remark)
	}
	if retained.Items[0].Usage == nil || retained.Items[0].Usage.Total != 6 {
		t.Fatalf("usage = %+v, want the stored counters to survive a resolve", retained.Items[0].Usage)
	}
}

func TestForget_DropsMetadataAndKeepsARecoveryCopy(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	remarkBody, err := json.Marshal(map[string]string{"key": "sk-retired", "remark": "old key"})
	if err != nil {
		t.Fatalf("encode remark: %v", err)
	}
	if status, _ := managementCall(t, rt, "PUT", management.RouteRemarks, remarkBody); status != 200 {
		t.Fatalf("remark status = %d, want 200", status)
	}
	usageCall(t, rt, map[string]any{"APIKey": "sk-retired", "Model": "gpt-5.6", "Detail": map[string]any{"TotalTokens": 6}})
	if status, _ := managementCall(t, rt, "POST", management.RouteResolve, []byte(`{"keys":["sk-retired"]}`)); status != 200 {
		t.Fatalf("resolve status = %d, want 200", status)
	}

	hash := remarks.Index("sk-retired")
	status, body := managementCall(t, rt, "POST", management.RouteForget, []byte(`{"hashes":["`+hash+`"]}`))
	if status != 200 {
		t.Fatalf("forget status = %d: %s", status, body)
	}
	var result management.ForgetResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode forget result: %v", err)
	}
	if result.Dropped != 1 {
		t.Fatalf("dropped = %d, want the one key that had metadata", result.Dropped)
	}

	after := resolveResult(t, rt, []string{"sk-retired"})
	if after.Items[0].Remark != "" || after.Items[0].Usage != nil {
		t.Fatalf("items[0] = %+v, want the key to keep no remark and no counters", after.Items[0])
	}
	buckets, err := state.LoadBuckets(BucketsPath(statePath))
	if err != nil {
		t.Fatalf("state.LoadBuckets() error = %v", err)
	}
	if _, ok := buckets.Buckets()[hash]; ok {
		t.Fatal("bucket file still holds usage for a forgotten key")
	}

	bucketBackup, err := state.LoadBuckets(BucketsPath(statePath) + ".bak")
	if err != nil {
		t.Fatalf("state.LoadBuckets(backup) error = %v", err)
	}
	if _, ok := bucketBackup.Buckets()[hash]; !ok {
		t.Fatal("bucket backup does not hold the counters that were dropped")
	}
	backup, err := state.Load(statePath + ".bak")
	if err != nil {
		t.Fatalf("state.Load(backup) error = %v", err)
	}
	if entry, ok := backup.Remark(hash); !ok || entry.Remark != "old key" {
		t.Fatalf("backup remark = %+v (ok=%v), want the dropped remark", entry, ok)
	}

	// Forgetting an unknown key is a no-op and must not overwrite the backup.
	status, body = managementCall(t, rt, "POST", management.RouteForget, []byte(`{"hashes":["`+strings.Repeat("0", 64)+`"]}`))
	if status != 200 {
		t.Fatalf("second forget status = %d: %s", status, body)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode second forget: %v", err)
	}
	if result.Dropped != 0 {
		t.Fatalf("second dropped = %d, want 0", result.Dropped)
	}
}

func TestResolve_SplitsUsageByWindow(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))
	now := time.Now().UTC().Truncate(time.Hour)

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": now.Add(-2 * time.Hour).Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 10}})
	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": now.Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 20}})

	recent := resolveWindow(t, rt, []string{"sk-a"}, now, now.Add(time.Hour))
	if recent.Items[0].Usage == nil || recent.Items[0].Usage.Total != 20 {
		t.Fatalf("recent window = %+v, want only the 20 tokens recorded this hour", recent.Items[0].Usage)
	}

	older := resolveWindow(t, rt, []string{"sk-a"}, now.Add(-2*time.Hour), now.Add(-time.Hour))
	if older.Items[0].Usage == nil || older.Items[0].Usage.Total != 10 {
		t.Fatalf("older window = %+v, want only the 10 tokens recorded two hours ago", older.Items[0].Usage)
	}

	both := resolveWindow(t, rt, []string{"sk-a"}, now.Add(-3*time.Hour), now.Add(time.Hour))
	if both.Items[0].Usage == nil || both.Items[0].Usage.Total != 30 {
		t.Fatalf("wide window = %+v, want both records", both.Items[0].Usage)
	}
}

func TestResolve_LastSeenIgnoresTheQueriedRange(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))
	now := time.Now().UTC().Truncate(time.Hour)
	older := now.Add(-48 * time.Hour).Add(17 * time.Minute)
	latest := now.Add(-9 * time.Minute)

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": older.Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 10}})
	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": latest.Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 20}})

	// The queried window only contains the older record, yet the lifetime last
	// call must still be the latest one.
	got := resolveWindow(t, rt, []string{"sk-a"}, now.Add(-72*time.Hour), now.Add(-24*time.Hour))
	if got.Items[0].Usage == nil || got.Items[0].Usage.Total != 10 {
		t.Fatalf("usage = %+v, want only the older record inside the window", got.Items[0].Usage)
	}
	if want := latest.Format(time.RFC3339); got.Items[0].LastSeen != want {
		t.Fatalf("last seen = %q, want the precise latest call %q", got.Items[0].LastSeen, want)
	}

	idle := resolveResult(t, rt, []string{"sk-unknown"})
	if idle.Items[0].LastSeen != "" {
		t.Fatalf("idle key last seen = %q, want empty", idle.Items[0].LastSeen)
	}
}

func TestResolve_ClampsWindowToRetention(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))
	now := time.Now().UTC()
	expired := now.AddDate(0, 0, -40).Truncate(time.Hour)

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": expired.Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 99}})

	if got := resolveWindow(t, rt, []string{"sk-a"}, expired, expired.Add(time.Hour)); got.Items[0].Usage != nil {
		t.Fatalf("usage outside retention = %+v, want none", got.Items[0].Usage)
	}
	if got := resolveResult(t, rt, []string{"sk-a"}); got.Items[0].Usage != nil {
		t.Fatalf("default window = %+v, want the retained window only", got.Items[0].Usage)
	}
}

// TestFlush_SplitsConfigurationAndBuckets pins the storage contract: the state
// document holds settings and remarks while every time bucket lives in its own
// file, and a bucket that fell out of retention is never written back.
func TestFlush_SplitsConfigurationAndBuckets(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)
	now := time.Now().UTC().Truncate(time.Hour)

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": now.Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 5}})
	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": now.AddDate(0, 0, -40).Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 500}})
	rt.FlushUsage()

	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if strings.Contains(string(raw), `"usage"`) || strings.Contains(string(raw), `"buckets"`) {
		t.Fatalf("state document still carries usage data:\n%s", raw)
	}

	bucketPath := BucketsPath(statePath)
	if bucketPath == statePath {
		t.Fatal("bucket path collides with the state path")
	}
	buckets, err := state.LoadBuckets(bucketPath)
	if err != nil {
		t.Fatalf("state.LoadBuckets() error = %v", err)
	}
	kept := buckets.Buckets()[remarks.Index("sk-a")]
	if len(kept) != 1 {
		t.Fatalf("buckets = %+v, want only the retained hour", kept)
	}
	if bucket := kept[usage.PeriodKey(now)]; bucket.Total != 5 {
		t.Fatalf("retained bucket = %+v, want the 5 tokens from now", bucket)
	}
}

// TestUpdateSettings_ShrinksRetentionImmediately pins requirement one: lowering
// the retention window drops the now-expired buckets at once, without waiting
// for the next request to dirty the aggregate.
func TestUpdateSettings_ShrinksRetentionImmediately(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)
	now := time.Now().UTC().Truncate(time.Hour)

	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": now.Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 5}})
	usageCall(t, rt, map[string]any{"APIKey": "sk-a", "Model": "m", "RequestedAt": now.AddDate(0, 0, -5).Format(time.RFC3339),
		"Detail": map[string]any{"TotalTokens": 50}})
	rt.FlushUsage()

	buckets, err := state.LoadBuckets(BucketsPath(statePath))
	if err != nil {
		t.Fatalf("state.LoadBuckets() error = %v", err)
	}
	if got := len(buckets.Buckets()[remarks.Index("sk-a")]); got != 2 {
		t.Fatalf("buckets before the change = %d, want 2", got)
	}

	if status, body := managementCall(t, rt, "PUT", management.RouteSettings, []byte(`{"retention_days":2}`)); status != 200 {
		t.Fatalf("settings status = %d: %s", status, body)
	}

	buckets, err = state.LoadBuckets(BucketsPath(statePath))
	if err != nil {
		t.Fatalf("state.LoadBuckets() after the change error = %v", err)
	}
	kept := buckets.Buckets()[remarks.Index("sk-a")]
	if len(kept) != 1 {
		t.Fatalf("buckets after shrinking retention = %+v, want only the current hour", kept)
	}
	if _, ok := kept[usage.PeriodKey(now)]; !ok {
		t.Fatalf("kept buckets = %+v, want the current hour", kept)
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
}

func TestUsageHandle_AcceptsSnakeCasePayload(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	// Backdated by two days so the record only answers a window that contains
	// its ReportedAt, which proves the snake_case field was parsed.
	requestedAt := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Hour)
	raw := fmt.Sprintf(`{"api_key":"sk-a","model":"gpt-5.6","failed":true,"requested_at":%q,`+
		`"detail":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}`, requestedAt.Format(time.RFC3339))
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodUsageHandle, []byte(raw)))
	if !env.OK {
		t.Fatalf("usage.handle failed: %+v", env.Error)
	}

	result := resolveWindow(t, rt, []string{"sk-a"}, requestedAt, requestedAt.Add(time.Hour))
	usageRecord := result.Items[0].Usage
	if usageRecord == nil || usageRecord.Input != 3 || usageRecord.Output != 4 || usageRecord.Total != 7 || usageRecord.Failed != 1 {
		t.Fatalf("usage = %+v, want the snake_case payload counters", usageRecord)
	}
	if outside := resolveWindow(t, rt, []string{"sk-a"}, requestedAt.Add(time.Hour), requestedAt.Add(2*time.Hour)); outside.Items[0].Usage != nil {
		t.Fatalf("usage outside the window = %+v, want none", outside.Items[0].Usage)
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
		reloaded, err := state.LoadBuckets(BucketsPath(statePath))
		if err == nil {
			for _, bucket := range reloaded.Buckets()[remarks.Index("sk-a")] {
				if bucket.Total == 42 {
					return
				}
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
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	restarted := registerRuntime(t, statePath)
	result := resolveResult(t, restarted, []string{"sk-a"})
	if result.Items[0].Usage == nil || result.Items[0].Usage.Total != 42 {
		t.Fatalf("usage after restart = %+v, want the persisted 42 tokens", result.Items[0].Usage)
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
func TestHandle_ManagementRegister_DeclaresAllRoutes(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodManagementRegister, nil))
	if !env.OK {
		t.Fatalf("management.register failed: %s", env.Result)
	}

	var reg struct {
		Routes []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(env.Result, &reg); err != nil {
		t.Fatalf("unmarshal registration: %v", err)
	}

	wanted := map[string]string{
		"GET " + management.RouteSettings:  "settings read",
		"PUT " + management.RouteSettings:  "settings write",
		"POST " + management.RouteResolve:  "resolve",
		"PUT " + management.RouteRemarks:   "remarks",
		"POST " + management.RouteForget:   "forget",
		"POST " + management.RouteGenerate: "generate",
		"POST " + management.RouteFailures: "failures",
		"GET " + management.RouteQuota:     "quota read",
		"PUT " + management.RouteQuota:     "quota write",
		"DELETE " + management.RouteQuota:  "quota delete",
		"GET " + management.RouteModels:    "models read",
	}

	declared := make(map[string]bool)
	for _, route := range reg.Routes {
		declared[route.Method+" "+route.Path] = true
	}

	for key, desc := range wanted {
		if !declared[key] {
			t.Errorf("missing route for %s: %s", desc, key)
		}
	}
}

func TestRuntime_ErrorLogRecordingAndQuery(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	// Push a failure usage record
	failureRecord := []byte(`{
		"APIKey": "sk-testkey",
		"Model": "gpt-5.6",
		"Failed": true,
		"RequestedAt": "2026-10-19T10:00:00Z",
		"Failure": {
			"StatusCode": 429,
			"Body": "rate limit exceeded"
		}
	}`)
	rt.Handle(context.Background(), MethodUsageHandle, failureRecord)

	// Query failures
	keyHash := remarks.Index("sk-testkey")
	res, err := rt.Failures(context.Background(), management.FailuresRequest{Hash: keyHash})
	if err != nil {
		t.Fatalf("Failures() error = %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 failure item, got %d", len(res.Items))
	}
	if res.Items[0].StatusCode != 429 || res.Items[0].Message != "rate limit exceeded" {
		t.Fatalf("unexpected failure entry: %+v", res.Items[0])
	}

	// Flush and verify errors file exists
	rt.FlushUsage()
	errorsPath := filepath.Join(filepath.Dir(statePath), "state-errors.json")
	if _, err := os.Stat(errorsPath); err != nil {
		t.Fatalf("expected errors file at %s: %v", errorsPath, err)
	}

	// Forget the key
	if _, err := rt.Forget(context.Background(), []string{keyHash}); err != nil {
		t.Fatalf("Forget() error = %v", err)
	}
	resAfter, err := rt.Failures(context.Background(), management.FailuresRequest{Hash: keyHash})
	if err != nil {
		t.Fatalf("Failures() after Forget error = %v", err)
	}
	if len(resAfter.Items) != 0 {
		t.Fatalf("expected 0 failure items after forget, got %d", len(resAfter.Items))
	}
}

func TestHandle_RequestInterceptBefore_QuotaEnforcement(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	apiKey := "sk-test-quota-key"
	keyHash := remarks.Index(apiKey)

	// 1. Initial intercept without quota rule -> allowed
	reqPayload, _ := json.Marshal(map[string]any{
		"headers": map[string][]string{
			"Authorization": {"Bearer " + apiKey},
		},
		"model": "gpt-5.6",
	})
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, reqPayload))
	if !env.OK {
		t.Fatalf("intercept_before failed: %+v", env.Error)
	}
	var resp RequestInterceptResponse
	if err := json.Unmarshal(env.Result, &resp); err != nil {
		t.Fatalf("unmarshal intercept response: %v", err)
	}
	if resp.Terminate {
		t.Fatal("expected terminate = false before quota rule configured")
	}

	// 2. Configure quota rule via Management API: 1000 tokens / daily
	quotaBody := fmt.Sprintf(`{"hash":"%s","enabled":true,"strategy":"token","window":"daily","limit_value":1000}`, keyHash)
	if status, _ := managementCall(t, rt, http.MethodPut, management.RouteQuota, []byte(quotaBody)); status != http.StatusOK {
		t.Fatalf("PUT quota status = %d, want 200", status)
	}

	// 3. Intercept before usage -> still allowed
	env = decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, reqPayload))
	if !env.OK {
		t.Fatalf("intercept_before failed: %+v", env.Error)
	}
	_ = json.Unmarshal(env.Result, &resp)
	if resp.Terminate {
		t.Fatal("expected terminate = false under quota")
	}

	// 4. Inject usage exceeding limit: 1200 tokens
	usageRecord := []byte(fmt.Sprintf(`{"APIKey":%q,"Model":"gpt-5.6","detail":{"total_tokens":1200}}`, apiKey))
	rt.Handle(context.Background(), MethodUsageHandle, usageRecord)

	// 5. Intercept after usage -> rejected with HTTP 403 (NOT 429!)
	env = decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, reqPayload))
	if !env.OK {
		t.Fatalf("intercept_before failed: %+v", env.Error)
	}
	var rejectedResp RequestInterceptResponse
	if err := json.Unmarshal(env.Result, &rejectedResp); err != nil {
		t.Fatalf("unmarshal intercept response: %v", err)
	}
	if !rejectedResp.Terminate {
		t.Fatal("expected terminate = true when quota exceeded")
	}
	if rejectedResp.StatusCode != http.StatusForbidden {
		t.Fatalf("status code = %d, want 403 Forbidden (must NOT be 429)", rejectedResp.StatusCode)
	}
	if !strings.Contains(rejectedResp.ResponseBody, "quota_exceeded") || !strings.Contains(rejectedResp.ResponseBody, "insufficient_quota") {
		t.Fatalf("response body missing quota error: %s", rejectedResp.ResponseBody)
	}

	// 6. Verify Resolve includes quota information
	resolvePayload := fmt.Sprintf(`{"keys":[%q]}`, apiKey)
	var resolveResult management.ResolveResult
	if status, body := managementCall(t, rt, http.MethodPost, management.RouteResolve, []byte(resolvePayload)); status != http.StatusOK {
		t.Fatalf("resolve status = %d, want 200: %s", status, body)
	} else if err := json.Unmarshal(body, &resolveResult); err != nil {
		t.Fatalf("decode resolve result: %v", err)
	}
	if len(resolveResult.Items) != 1 || resolveResult.Items[0].Quota == nil {
		t.Fatalf("resolve items missing quota summary: %+v", resolveResult)
	}
	if !resolveResult.Items[0].Quota.Exceeded {
		t.Fatal("expected quota.Exceeded = true")
	}
	if resolveResult.Items[0].Quota.UsedValue != 1200 {
		t.Fatalf("used value = %d, want 1200", resolveResult.Items[0].Quota.UsedValue)
	}

	// 7. Delete quota rule via Management API -> allowed again
	if status, _ := managementCall(t, rt, http.MethodDelete, management.RouteQuota+"?hash="+keyHash, nil); status != http.StatusOK {
		t.Fatalf("DELETE quota status = %d, want 200", status)
	}
	env = decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, reqPayload))
	_ = json.Unmarshal(env.Result, &resp)
	if resp.Terminate {
		t.Fatal("expected terminate = false after quota rule deleted")
	}
}

func TestHandle_RequestInterceptBefore_AllowedModels(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntime(t, statePath)

	apiKey := "sk-test-model-key"
	keyHash := remarks.Index(apiKey)

	// Configure quota rule with allowed_models = ["gpt-5.6"]
	quotaBody := fmt.Sprintf(`{"hash":"%s","enabled":true,"strategy":"token","window":"daily","limit_value":100000,"allowed_models":["gpt-5.6"]}`, keyHash)
	if status, _ := managementCall(t, rt, http.MethodPut, management.RouteQuota, []byte(quotaBody)); status != http.StatusOK {
		t.Fatalf("PUT quota status = %d, want 200", status)
	}

	// 1. Request allowed model -> permitted
	reqAllowed, _ := json.Marshal(map[string]any{
		"headers": map[string][]string{"Authorization": {"Bearer " + apiKey}},
		"model":   "gpt-5.6",
	})
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, reqAllowed))
	var resp RequestInterceptResponse
	_ = json.Unmarshal(env.Result, &resp)
	if resp.Terminate {
		t.Fatal("expected terminate = false for whitelisted model")
	}

	// 2. Request unallowed model -> rejected with 403 (NOT 429!)
	reqDenied, _ := json.Marshal(map[string]any{
		"headers": map[string][]string{"Authorization": {"Bearer " + apiKey}},
		"model":   "claude-3-7-sonnet",
	})
	env = decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, reqDenied))
	var respDenied RequestInterceptResponse
	_ = json.Unmarshal(env.Result, &respDenied)
	if !respDenied.Terminate {
		t.Fatal("expected terminate = true for unallowed model")
	}
	if respDenied.StatusCode != http.StatusForbidden {
		t.Fatalf("status code = %d, want 403 Forbidden", respDenied.StatusCode)
	}
	if !strings.Contains(respDenied.ResponseBody, "not permitted") {
		t.Fatalf("response body = %s, want not permitted error", respDenied.ResponseBody)
	}
}
