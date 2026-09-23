package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/state"
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
	rt := New(Options{StatePath: statePath})
	request := fmt.Sprintf(`{"config_yaml":%q}`, "state_path: "+filepath.ToSlash(statePath)+"\n")
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodPluginRegister, []byte(request)))
	if !env.OK {
		t.Fatalf("register failed: %s", env.Result)
	}
	return rt
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
	if result.Metadata.Version != PluginVersion {
		t.Fatalf("metadata version = %q, want %q", result.Metadata.Version, PluginVersion)
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
