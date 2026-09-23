// Package management owns the plugin HTTP surface: the plugin-owned Management
// API routes and the browser resource page. It depends on the runtime through
// the Backend interface only.
package management

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cpa-secret-manager/internal/usage"
)

const (
	// PluginID is the plugin identifier, derived by the host from the dynamic
	// library file name and used in configuration and route paths.
	PluginID = "cpa-secret-manager"
	// MenuLabel is the management center menu entry for the resource page.
	MenuLabel = "API Key Manager"

	managementBase = "/v0/management/plugins/" + PluginID
	resourceBase   = "/v0/resource/plugins/" + PluginID
	legacyBase     = "/plugins/" + PluginID

	// PathSettings is the plugin-owned settings route.
	PathSettings = "/settings"
	// PathResolve returns remark and usage data aligned with a key list.
	PathResolve = "/resolve"
	// PathRemarks creates, replaces or clears one remark.
	PathRemarks = "/remarks"
	// PathGenerate returns a freshly generated proxy API key.
	PathGenerate = "/keys/generate"

	// RouteSettings is the settings route path registered with the host.
	RouteSettings = legacyBase + PathSettings
	// RouteResolve is the resolve route path registered with the host.
	RouteResolve = legacyBase + PathResolve
	// RouteRemarks is the remark route path registered with the host.
	RouteRemarks = legacyBase + PathRemarks
	// RouteGenerate is the key generation route path registered with the host.
	RouteGenerate = legacyBase + PathGenerate

	// maxRequestBody bounds plugin management request bodies.
	maxRequestBody = 1 << 20
)

// Settings is the plugin business configuration exposed to the page.
type Settings struct {
	UsageEnabled   bool     `json:"usage_enabled"`
	StatePath      string   `json:"state_path"`
	StateWarning   string   `json:"state_warning,omitempty"`
	UsageWarning   string   `json:"usage_warning,omitempty"`
	ConfigWarnings []string `json:"config_warnings,omitempty"`
}

// SettingsUpdate is the request payload for updating business settings.
type SettingsUpdate struct {
	UsageEnabled *bool `json:"usage_enabled"`
}

// KeyEntry is the per-key payload returned by Resolve. Items are aligned with
// the submitted key list.
type KeyEntry struct {
	Hash   string `json:"hash"`
	Remark string `json:"remark"`
	// Usage is nil until the key served at least one attributable request.
	Usage *UsageSummary `json:"usage,omitempty"`
}

// UsageSummary is the cumulative token accounting shown for one key.
type UsageSummary struct {
	usage.Counters
	FirstSeen string       `json:"first_seen,omitempty"`
	LastSeen  string       `json:"last_seen,omitempty"`
	Models    []ModelUsage `json:"models,omitempty"`
}

// ModelUsage is the cumulative token accounting for one model of one key.
type ModelUsage struct {
	Model string `json:"model"`
	usage.Counters
}

// ResolveRequest carries the current proxy key list, in display order.
type ResolveRequest struct {
	Keys []string `json:"keys"`
}

// ResolveResult is the resolved metadata for the submitted key list.
type ResolveResult struct {
	Items []KeyEntry `json:"items"`
	// UnattributedRequests counts usage records that carried no attributable
	// key, so the page can explain a key that shows no usage.
	UnattributedRequests int64 `json:"unattributed_requests"`
	// OrphanRemarks counts stored remarks whose key is absent from the submitted
	// list, so the page can explain leftover metadata after a key is removed.
	OrphanRemarks int `json:"orphan_remarks"`
	// OrphanUsage counts usage aggregates whose key is absent from the submitted
	// list.
	OrphanUsage int `json:"orphan_usage"`
}

// RemarkUpdate sets or clears the remark of one proxy API key.
type RemarkUpdate struct {
	Key    string `json:"key"`
	Remark string `json:"remark"`
}

// GenerateResult carries one generated proxy API key.
type GenerateResult struct {
	Key string `json:"key"`
}

// InvalidRequest reports a request rejected by business validation. The HTTP
// adapter maps it to status 400.
type InvalidRequest struct {
	Message string
}

func (e *InvalidRequest) Error() string { return e.Message }

// NewInvalidRequest builds an InvalidRequest error.
func NewInvalidRequest(message string) error {
	return &InvalidRequest{Message: message}
}

// Backend is the runtime surface the HTTP adapter is allowed to call.
type Backend interface {
	Settings(ctx context.Context) (Settings, error)
	UpdateSettings(ctx context.Context, usageEnabled bool) error
	Resolve(ctx context.Context, apiKeys []string) (ResolveResult, error)
	SetRemark(ctx context.Context, apiKey string, remark string) error
	GenerateKey(ctx context.Context) (string, error)
}

// Handler serves the plugin-owned Management API routes.
type Handler struct {
	backend Backend
}

// NewHandler constructs the handler for the given backend.
func NewHandler(backend Backend) *Handler {
	return &Handler{backend: backend}
}

// ServeHTTP dispatches one plugin HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || w == nil || r == nil || r.URL == nil {
		http.Error(w, "handler unavailable", http.StatusInternalServerError)
		return
	}
	if h.backend == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorBody("backend_unavailable", "runtime not available"))
		return
	}

	switch normalizePath(r.URL.Path) {
	case PathSettings:
		h.serveSettings(w, r)
	case PathResolve:
		h.serveResolve(w, r)
	case PathRemarks:
		h.serveRemarks(w, r)
	case PathGenerate:
		h.serveGenerate(w, r)
	default:
		writeJSON(w, http.StatusNotFound, errorBody("not_found", "unknown plugin route"))
	}
}

func (h *Handler) serveSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := h.backend.Settings(r.Context())
		if h.fail(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut, http.MethodPatch:
		var update SettingsUpdate
		if !decodeBody(w, r, &update) {
			return
		}
		if update.UsageEnabled == nil {
			writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "usage_enabled is required"))
			return
		}
		if h.fail(w, h.backend.UpdateSettings(r.Context(), *update.UsageEnabled)) {
			return
		}
		writeJSON(w, http.StatusOK, statusBody())
	default:
		methodNotAllowed(w, "GET, PUT, PATCH")
	}
}

func (h *Handler) serveResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request ResolveRequest
	if !decodeBody(w, r, &request) {
		return
	}
	result, err := h.backend.Resolve(r.Context(), request.Keys)
	if h.fail(w, err) {
		return
	}
	if result.Items == nil {
		result.Items = []KeyEntry{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) serveRemarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w, http.MethodPut)
		return
	}
	var update RemarkUpdate
	if !decodeBody(w, r, &update) {
		return
	}
	if h.fail(w, h.backend.SetRemark(r.Context(), update.Key, update.Remark)) {
		return
	}
	writeJSON(w, http.StatusOK, statusBody())
}

func (h *Handler) serveGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	key, err := h.backend.GenerateKey(r.Context())
	if h.fail(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, GenerateResult{Key: key})
}

// fail writes an error response and reports whether the request is finished.
func (h *Handler) fail(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var invalid *InvalidRequest
	if errors.As(err, &invalid) {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", invalid.Message))
		return true
	}
	writeJSON(w, http.StatusInternalServerError, errorBody("backend_failed", err.Error()))
	return true
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "request body must be a JSON object"))
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeJSON(w, http.StatusMethodNotAllowed, errorBody("method_not_allowed", "method not allowed"))
}

// normalizePath reduces a host-forwarded path to the plugin-relative route.
func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	for _, prefix := range []string{managementBase, resourceBase, legacyBase} {
		if trimmed == prefix {
			return "/"
		}
		if strings.HasPrefix(trimmed, prefix+"/") {
			trimmed = strings.TrimPrefix(trimmed, prefix)
			break
		}
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "response encode failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func errorBody(code string, message string) map[string]string {
	return map[string]string{"error": code, "message": message}
}

func statusBody() map[string]string {
	return map[string]string{"status": "ok"}
}

// Recorder is a minimal http.ResponseWriter used to run the handler inside the
// CPA management bridge without importing net/http/httptest in production code.
type Recorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

// NewRecorder returns an empty recorder.
func NewRecorder() *Recorder {
	return &Recorder{header: make(http.Header)}
}

// Header returns the response header map.
func (rec *Recorder) Header() http.Header {
	if rec.header == nil {
		rec.header = make(http.Header)
	}
	return rec.header
}

// Write appends response bytes, defaulting the status to 200.
func (rec *Recorder) Write(data []byte) (int, error) {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	return rec.body.Write(data)
}

// WriteHeader records the response status.
func (rec *Recorder) WriteHeader(status int) {
	if rec.status == 0 {
		rec.status = status
	}
}

// Result returns the recorded status, headers and body.
func (rec *Recorder) Result() (int, http.Header, []byte) {
	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}
	return status, rec.header.Clone(), rec.body.Bytes()
}
