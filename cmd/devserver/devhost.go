package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cpa-secret-manager/internal/management"
	pluginruntime "cpa-secret-manager/internal/runtime"
	"cpa-secret-manager/internal/version"
)

// devHost simulates the CLIProxyAPI side of the plugin contract: the proxy API
// key store, the authenticated management API, the plugin resource menu and the
// usage record stream. It reuses the production runtime and page assets.
type devHost struct {
	managementKey string
	keysPath      string
	statePath     string
	shellHTML     string
	runtime       *pluginruntime.Runtime

	mu   sync.Mutex
	keys []string
}

type devHostOptions struct {
	ManagementKey string
	KeysPath      string
	StatePath     string
}

func newDevHost(options devHostOptions) (*devHost, error) {
	keys, err := loadKeyList(options.KeysPath)
	if err != nil {
		return nil, err
	}
	host := &devHost{
		managementKey: options.ManagementKey,
		keysPath:      options.KeysPath,
		statePath:     options.StatePath,
		shellHTML:     hostShellHTML(options.ManagementKey),
		runtime:       pluginruntime.New(pluginruntime.Options{StatePath: options.StatePath}),
		keys:          keys,
	}
	if err := host.register(); err != nil {
		return nil, err
	}
	return host, nil
}

// register performs the host handshake against the production runtime.
func (h *devHost) register() error {
	payload, err := json.Marshal(map[string]string{
		"config_yaml": "state_path: " + filepath.ToSlash(h.statePath) + "\n",
	})
	if err != nil {
		return err
	}
	raw := h.runtime.Handle(context.Background(), pluginruntime.MethodPluginRegister, payload)
	var envelope struct {
		OK    bool `json:"ok"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode plugin.register response: %w", err)
	}
	if !envelope.OK {
		return fmt.Errorf("plugin.register failed: %+v", envelope.Error)
	}
	return nil
}

func (h *devHost) shutdown(ctx context.Context) error {
	return h.runtime.Shutdown(ctx)
}

func (h *devHost) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("GET /control-panel", h.handleControlPanel)
	mux.HandleFunc("POST /dev/simulate-usage", h.handleSimulateUsage)
	mux.HandleFunc("POST /dev/seed", h.handleSeed)
	mux.HandleFunc("/v0/management/", h.handleManagement)
	mux.HandleFunc("/v0/resource/", h.handleResource)
	return mux
}

func (h *devHost) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/control-panel", http.StatusFound)
}

func (h *devHost) handleControlPanel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.shellHTML))
}

// handleManagement mirrors the host middleware: every /v0/management request
// needs the management key before either the simulated host routes or the
// plugin-owned routes answer.
func (h *devHost) handleManagement(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid management key"})
		return
	}
	switch r.URL.Path {
	case "/v0/management/api-keys":
		h.serveAPIKeys(w, r)
		return
	case "/v0/management/plugins":
		h.servePluginList(w, r)
		return
	}
	h.runtime.ManagementHandler().ServeHTTP(w, r)
}

// handleResource serves browser resources without management authentication,
// exactly like the host does for /v0/resource/plugins/<pluginID>/<path>.
func (h *devHost) handleResource(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/v0/resource/plugins/") {
		http.NotFound(w, r)
		return
	}
	h.runtime.ManagementHandler().ServeHTTP(w, r)
}

func (h *devHost) authorized(r *http.Request) bool {
	if h.managementKey == "" {
		return true
	}
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		header = strings.TrimSpace(header[7:])
	}
	if header == h.managementKey {
		return true
	}
	return r.Header.Get("X-Management-Key") == h.managementKey
}

func (h *devHost) serveAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string][]string{"api-keys": h.keyList()})
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		next, changed, err := h.mutateKeys(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if !changed {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if err := h.replaceKeys(next); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		w.Header().Set("Allow", "GET, PUT, PATCH, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *devHost) mutateKeys(r *http.Request) ([]string, bool, error) {
	current := h.keyList()
	query := r.URL.Query()

	if r.Method == http.MethodDelete {
		if value := query.Get("value"); value != "" {
			next := make([]string, 0, len(current))
			for _, key := range current {
				if key != value {
					next = append(next, key)
				}
			}
			return next, len(next) != len(current), nil
		}
		index, err := strconv.Atoi(query.Get("index"))
		if err != nil || index < 0 || index >= len(current) {
			return nil, false, fmt.Errorf("index is required for DELETE")
		}
		return append(append([]string{}, current[:index]...), current[index+1:]...), true, nil
	}

	var body []byte
	if r.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			return nil, false, err
		}
		body = raw
	}

	if r.Method == http.MethodPut {
		next, err := decodeKeyList(body)
		if err != nil {
			return nil, false, err
		}
		return next, true, nil
	}

	// PATCH supports both old/new and index/value, like the host.
	var patch struct {
		Old   string `json:"old"`
		New   string `json:"new"`
		Value string `json:"value"`
		Index *int   `json:"index"`
	}
	if err := json.Unmarshal(body, &patch); err != nil {
		return nil, false, fmt.Errorf("decode patch body: %w", err)
	}
	switch {
	case patch.Index != nil:
		if *patch.Index < 0 || *patch.Index >= len(current) {
			return nil, false, fmt.Errorf("index out of range")
		}
		current[*patch.Index] = patch.Value
		return current, true, nil
	case patch.Old != "":
		for index, key := range current {
			if key == patch.Old {
				current[index] = patch.New
				return current, true, nil
			}
		}
		return nil, false, fmt.Errorf("old value not found")
	default:
		return nil, false, fmt.Errorf("patch requires old/new or index/value")
	}
}

func (h *devHost) servePluginList(w http.ResponseWriter, r *http.Request) {
	type menu struct {
		Path        string `json:"path"`
		Menu        string `json:"menu"`
		Description string `json:"description"`
	}
	type metadata struct {
		Name             string `json:"name"`
		Version          string `json:"version"`
		Author           string `json:"author"`
		GitHubRepository string `json:"githubRepository"`
		Logo             string `json:"logo"`
	}
	type entry struct {
		ID               string   `json:"id"`
		Path             string   `json:"path"`
		Configured       bool     `json:"configured"`
		Registered       bool     `json:"registered"`
		Enabled          bool     `json:"enabled"`
		EffectiveEnabled bool     `json:"effectiveEnabled"`
		SupportsOAuth    bool     `json:"supportsOAuth"`
		Logo             string   `json:"logo"`
		Menus            []menu   `json:"menus"`
		Metadata         metadata `json:"metadata"`
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pluginsEnabled": true,
		"pluginsDir":     "plugins",
		"plugins": []entry{{
			ID:               management.PluginID,
			Path:             "plugins/" + management.PluginID + ".dll",
			Configured:       true,
			Registered:       true,
			Enabled:          true,
			EffectiveEnabled: true,
			Menus: []menu{{
				Path:        "/v0/resource/plugins/" + management.PluginID + management.PathPage,
				Menu:        management.MenuLabel,
				Description: management.PageDescription,
			}},
			Metadata: metadata{
				Name:             "CPA Secret Manager",
				Version:          version.PluginVersion,
				Author:           "ygq-future",
				GitHubRepository: "https://github.com/ygq-future/cpa-secret-manager",
			},
		}},
	})
}

// handleSimulateUsage pushes one or more host-shaped usage records through the
// production usage.handle path.
func (h *devHost) handleSimulateUsage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	apiKey := query.Get("key")
	if apiKey == "" {
		// Usage is only ever attributed to a managed key; a record without one is
		// not accounted anywhere.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}
	model := query.Get("model")
	if model == "" {
		model = "gpt-5.6"
	}
	count := intQuery(query.Get("count"), 1)
	if count < 1 || count > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "count must be between 1 and 1000"})
		return
	}
	failed := query.Get("failed") == "true"

	for range count {
		record := map[string]any{
			"Provider":    "openai",
			"APIKey":      apiKey,
			"Model":       model,
			"Failed":      failed,
			"RequestedAt": time.Now().UTC().Format(time.RFC3339Nano),
			"Detail": map[string]any{
				"InputTokens":     intQuery(query.Get("input"), 120),
				"OutputTokens":    intQuery(query.Get("output"), 240),
				"ReasoningTokens": intQuery(query.Get("reasoning"), 0),
				"CachedTokens":    intQuery(query.Get("cached"), 0),
				"CacheReadTokens": intQuery(query.Get("cache_read"), 0),
				"TotalTokens":     intQuery(query.Get("total"), 360),
			},
		}
		raw, err := json.Marshal(record)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		h.runtime.Handle(context.Background(), pluginruntime.MethodUsageHandle, raw)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "count": count, "model": model})
}

// handleSeed prepares a demonstrable dataset: proxy keys, remarks and usage.
func (h *devHost) handleSeed(w http.ResponseWriter, r *http.Request) {
	count := intQuery(r.URL.Query().Get("keys"), 3)
	if count < 1 || count > 20 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "keys must be between 1 and 20"})
		return
	}

	current := h.keyList()
	seeded := make([]string, 0, count)
	for range count {
		key, err := h.runtime.GenerateKey(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		current = append(current, key)
		seeded = append(seeded, key)
	}
	if err := h.replaceKeys(current); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	models := []string{"gpt-5.6", "claude-sonnet", "gemini-3-flash"}
	for index, key := range seeded {
		remark := fmt.Sprintf("dev key %d", index+1)
		if err := h.runtime.SetRemark(r.Context(), key, remark); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for modelIndex, model := range models {
			payload, err := json.Marshal(map[string]any{
				"Provider":    "openai",
				"APIKey":      key,
				"Model":       model,
				"Failed":      modelIndex == 2 && index%2 == 0,
				"RequestedAt": time.Now().UTC().Format(time.RFC3339Nano),
				"Detail": map[string]any{
					"InputTokens":     100 * (modelIndex + 1),
					"OutputTokens":    200 * (modelIndex + 1),
					"ReasoningTokens": 10 * (modelIndex + 1),
					"CachedTokens":    5 * (modelIndex + 1),
					"TotalTokens":     310 * (modelIndex + 1),
				},
			})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			h.runtime.Handle(context.Background(), pluginruntime.MethodUsageHandle, payload)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "keys": seeded})
}

func (h *devHost) keyList() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.keys...)
}

func (h *devHost) replaceKeys(keys []string) error {
	h.mu.Lock()
	h.keys = append([]string(nil), keys...)
	snapshot := append([]string(nil), h.keys...)
	h.mu.Unlock()
	return saveKeyList(h.keysPath, snapshot)
}

func loadKeyList(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read key list %s: %w", path, err)
	}
	return decodeKeyList(raw)
}

func saveKeyList(path string, keys []string) error {
	encoded, err := json.MarshalIndent(map[string][]string{"api-keys": keys}, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(encoded, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// decodeKeyList accepts every list shape the host and this simulator use: a raw
// array, the GET response envelope and the wrapped items object.
func decodeKeyList(body []byte) ([]string, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("request body is required")
	}
	if strings.HasPrefix(trimmed, "[") {
		var keys []string
		if err := json.Unmarshal([]byte(trimmed), &keys); err != nil {
			return nil, fmt.Errorf("decode key list: %w", err)
		}
		return keys, nil
	}
	var wrapped struct {
		APIKeys []string `json:"api-keys"`
		Items   []string `json:"items"`
	}
	if err := json.Unmarshal([]byte(trimmed), &wrapped); err != nil {
		return nil, fmt.Errorf("decode key list: %w", err)
	}
	if wrapped.APIKeys != nil {
		return wrapped.APIKeys, nil
	}
	return wrapped.Items, nil
}

func intQuery(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "encode failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
