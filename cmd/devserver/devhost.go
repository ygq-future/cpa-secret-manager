package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// latency delays every management response so the page's loading states are
	// observable; a local host answers in microseconds, which paints no frame.
	latency time.Duration

	mu   sync.Mutex
	keys []string
}

type devHostOptions struct {
	ManagementKey string
	KeysPath      string
	StatePath     string
	Latency       time.Duration
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
		latency:       options.Latency,
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
	mux.HandleFunc("POST /dev/simulate-intercept", h.handleSimulateIntercept)
	mux.HandleFunc("POST /dev/seed", h.handleSeed)
	mux.HandleFunc("POST /dev/seed-history", h.handleSeedHistory)
	mux.HandleFunc("POST /dev/reset", h.handleReset)
	mux.HandleFunc("GET /v1/models", h.handleV1Models)
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
	if !h.delay(r) {
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

// delay holds the response for the configured latency so the page's spinners and
// table masks stay on screen long enough to be seen. It reports false when the
// caller went away while waiting.
func (h *devHost) delay(r *http.Request) bool {
	if h.latency <= 0 {
		return true
	}
	timer := time.NewTimer(h.latency)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-r.Context().Done():
		return false
	}
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
	statusCode := intQuery(query.Get("status_code"), 0)
	errorBody := query.Get("error_body")
	if errorBody == "" {
		errorBody = query.Get("message")
	}
	if !failed && statusCode >= 400 {
		failed = true
	}
	if failed && statusCode == 0 {
		statusCode = 500
	}
	if failed && errorBody == "" {
		errorBody = "Simulated upstream failure"
	}
	at, err := resolveRecordTime(query)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	detail := usageDetail{
		Input:     intQuery(query.Get("input"), 120),
		Output:    intQuery(query.Get("output"), 240),
		Reasoning: intQuery(query.Get("reasoning"), 0),
		Cached:    intQuery(query.Get("cached"), 0),
		CacheRead: intQuery(query.Get("cache_read"), 0),
		Total:     intQuery(query.Get("total"), 360),
	}
	for range count {
		h.injectUsage(apiKey, model, at, failed, statusCode, errorBody, detail)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "count": count, "model": model, "at": at.Format(time.RFC3339)})
}

// resolveRecordTime turns the optional "at" (RFC3339) and "days_ago" (float)
// query parameters into the instant the injected records carry. Without either
// the record is stamped "now", which is why a default seed only ever populates
// today; both parameters exist so the simulator can produce history that a
// date-filtered view can actually exercise.
func resolveRecordTime(query url.Values) (time.Time, error) {
	if raw := strings.TrimSpace(query.Get("at")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, fmt.Errorf("at must be RFC3339: %v", err)
		}
		return parsed.UTC(), nil
	}
	if raw := strings.TrimSpace(query.Get("days_ago")); raw != "" {
		days, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("days_ago must be a number: %v", err)
		}
		return time.Now().UTC().Add(-time.Duration(days * float64(24*time.Hour))), nil
	}
	return time.Now().UTC(), nil
}

// usageDetail mirrors the token accounting the host reports for one request.
type usageDetail struct {
	Input     int
	Output    int
	Reasoning int
	Cached    int
	CacheRead int
	Total     int
}

// injectUsage pushes one host-shaped usage record through the production usage
// path, stamped with at.
func (h *devHost) injectUsage(apiKey, model string, at time.Time, failed bool, statusCode int, message string, detail usageDetail) {
	record := map[string]any{
		"Provider":    "openai",
		"APIKey":      apiKey,
		"Model":       model,
		"Failed":      failed,
		"RequestedAt": at.UTC().Format(time.RFC3339Nano),
		"Detail": map[string]any{
			"InputTokens":     detail.Input,
			"OutputTokens":    detail.Output,
			"ReasoningTokens": detail.Reasoning,
			"CachedTokens":    detail.Cached,
			"CacheReadTokens": detail.CacheRead,
			"TotalTokens":     detail.Total,
		},
	}
	if failed {
		record["Failure"] = map[string]any{
			"StatusCode": statusCode,
			"Body":       message,
		}
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return
	}
	h.runtime.Handle(context.Background(), pluginruntime.MethodUsageHandle, raw)
}

// handleSeedHistory fills the simulator with a day-by-day history so a
// date-filtered view has something to slice: every key gets usage on each of the
// last "days" calendar days, with a failure every third day. Unlike /dev/seed,
// whose records are all stamped "now", this endpoint makes the date range
// observable.
func (h *devHost) handleSeedHistory(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	keyCount := intQuery(query.Get("keys"), 3)
	if keyCount < 1 || keyCount > 20 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "keys must be between 1 and 20"})
		return
	}
	days := intQuery(query.Get("days"), 30)
	if days < 1 || days > 365 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be between 1 and 365"})
		return
	}
	if query.Get("reset") != "false" {
		if err := h.resetState(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	seeded := make([]string, 0, keyCount)
	for range keyCount {
		key, err := h.runtime.GenerateKey(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		seeded = append(seeded, key)
	}
	if err := h.replaceKeys(append(h.keyList(), seeded...)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	models := []string{"gpt-5.6", "claude-sonnet", "gemini-3-flash"}
	base := time.Now()
	records := 0
	failures := 0
	for index, key := range seeded {
		if err := h.runtime.SetRemark(r.Context(), key, fmt.Sprintf("dev key %d", index+1)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for day := 0; day < days; day++ {
			requests := 1 + (index+day)%3
			for n := 0; n < requests; n++ {
				model := models[(index+day+n)%len(models)]
				h.injectUsage(key, model, historyStamp(base, day, time.Duration(n)*time.Minute), false, 0, "", usageDetail{
					Input:     100 * (n + 1),
					Output:    200 * (n + 1),
					Reasoning: 10 * (n + 1),
					Cached:    5 * (n + 1),
					CacheRead: 7 * (n + 1),
					Total:     310 * (n + 1),
				})
				records++
			}
			if (day+index)%3 == 0 {
				h.injectUsage(key, models[(index+day)%len(models)], historyStamp(base, day, 90*time.Second), true, 429, "Rate limit exceeded for token quota", usageDetail{})
				failures++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"keys":     seeded,
		"days":     days,
		"records":  records,
		"failures": failures,
	})
}

// historyStamp places one record on the calendar day daysAgo back, offset
// earlier from the seeding instant. Every stamp is nudged backwards and clamped
// to its own midnight so a record can never land in the future: a "today"
// record in the future would be invisible to any range ending at now, which
// makes the date filter look broken when it is not.
func historyStamp(base time.Time, daysAgo int, offset time.Duration) time.Time {
	day := base.AddDate(0, 0, -daysAgo)
	at := day.Add(-offset)
	if start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()); at.Before(start) {
		return start
	}
	return at
}

// handleReset drops the simulated key list and the plugin state document, then
// rebuilds the runtime so the page starts from an empty dataset.
func (h *devHost) handleReset(w http.ResponseWriter, r *http.Request) {
	if err := h.resetState(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := h.replaceKeys(nil); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// resetState shuts the runtime down, removes its state and error-log documents
// and starts a fresh runtime against the same paths.
func (h *devHost) resetState() error {
	_ = h.runtime.Shutdown(context.Background())
	for _, path := range []string{h.statePath, pluginruntime.ErrorsPath(h.statePath), pluginruntime.BucketsPath(h.statePath), pluginruntime.BucketsPath(h.statePath) + ".bak"} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	h.runtime = pluginruntime.New(pluginruntime.Options{StatePath: h.statePath})
	return h.register()
}

func (h *devHost) handleSimulateIntercept(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		key = r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(key), "bearer ") {
			key = strings.TrimSpace(key[7:])
		}
	}
	model := r.URL.Query().Get("model")
	if model == "" {
		model = "gpt-5.6"
	}
	reqPayload, _ := json.Marshal(map[string]any{
		"headers": map[string][]string{
			"Authorization": {"Bearer " + key},
		},
		"model": model,
	})
	raw := h.runtime.Handle(context.Background(), pluginruntime.MethodRequestInterceptBefore, reqPayload)
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !envelope.OK {
		writeJSON(w, http.StatusInternalServerError, envelope.Error)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(envelope.Result)
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

func (h *devHost) handleV1Models(w http.ResponseWriter, r *http.Request) {
	models := []string{
		"gpt-5.6",
		"gpt-5.5",
		"claude-3-7-sonnet",
		"claude-3-5-sonnet",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
		"deepseek-r1",
	}
	type modelItem struct {
		ID     string `json:"id"`
		Object string `json:"object"`
	}
	list := make([]modelItem, 0, len(models))
	for _, m := range models {
		list = append(list, modelItem{ID: m, Object: "model"})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   list,
	})
}
