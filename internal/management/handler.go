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

	"cpa-secret-manager/internal/errorslog"
	"cpa-secret-manager/internal/quota"
	"cpa-secret-manager/internal/remarks"
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

	// PathPage is the browser resource path served to the management center.
	PathPage = "/keys"
	// PageDescription is the management center menu description.
	PageDescription = "Manage proxy API keys with per-key remarks and token usage."
	// PathSettings is the plugin-owned settings route.
	PathSettings = "/settings"
	// PathResolve returns remark and usage data aligned with a key list.
	PathResolve = "/resolve"
	// PathRemarks creates, replaces or clears one remark.
	PathRemarks = "/remarks"
	// PathForget drops the plugin-side metadata of the submitted keys.
	PathForget = "/forget"
	// PathGenerate returns a freshly generated proxy API key.
	PathGenerate = "/keys/generate"
	// PathFailures queries failure error logs by key hash and time range.
	PathFailures = "/failures"
	PathQuota    = "/quota"
	// PathModels returns known model names.
	PathModels    = "/models"
	RouteSettings = legacyBase + PathSettings
	// RouteResolve is the resolve route path registered with the host.
	RouteResolve = legacyBase + PathResolve
	// RouteRemarks is the remark route path registered with the host.
	RouteRemarks = legacyBase + PathRemarks
	// RouteForget is the metadata removal route path registered with the host.
	RouteForget = legacyBase + PathForget
	// RouteGenerate is the key generation route path registered with the host.
	RouteGenerate = legacyBase + PathGenerate
	// RouteFailures is the failures route path registered with the host.
	RouteFailures = legacyBase + PathFailures
	RouteQuota    = legacyBase + PathQuota
	// RouteModels is the models route registered with the host.
	RouteModels    = legacyBase + PathModels
	maxRequestBody = 1 << 20
)

// CacheFile describes a cache or state file on disk and its probed size.
type CacheFile struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Exists    bool   `json:"exists"`
}

// Settings is the plugin business configuration exposed to the page.
type Settings struct {
	UsageEnabled   bool        `json:"usage_enabled"`
	RetentionDays  int         `json:"retention_days"`
	StatePath      string      `json:"state_path"`
	StateWarning   string      `json:"state_warning,omitempty"`
	UsageWarning   string      `json:"usage_warning,omitempty"`
	ConfigWarnings []string    `json:"config_warnings,omitempty"`
	CacheFiles     []CacheFile `json:"cache_files,omitempty"`
}

// SettingsUpdate is the request payload for updating business settings.
type SettingsUpdate struct {
	UsageEnabled  *bool `json:"usage_enabled"`
	RetentionDays *int  `json:"retention_days"`
}

// FailuresRequest specifies query parameters for error log retrieval.
type FailuresRequest struct {
	Hash  string `json:"hash"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// FailuresResult returns the matched error log entries.
type FailuresResult struct {
	Items []errorslog.ErrorEntry `json:"items"`
}

// KeyEntry is the per-key payload returned by Resolve. Items are aligned with
// the submitted key list.
type KeyEntry struct {
	Hash   string `json:"hash"`
	Remark string `json:"remark"`
	// LastSeen is the exact instant of the key's latest observed request inside
	// the retention window, independent of the queried range.
	LastSeen string `json:"last_seen,omitempty"`
	// Usage is nil until the key served at least one attributable request.
	Usage *UsageSummary `json:"usage,omitempty"`
	// Quota describes the rule and window usage if configured.
	Quota *QuotaSummary `json:"quota,omitempty"`
}

// QuotaSummary describes a key's quota configuration and consumption state.
type QuotaSummary struct {
	Enabled       bool               `json:"enabled"`
	Strategy      quota.StrategyType `json:"strategy"`
	Window        quota.WindowType   `json:"window"`
	DurationSec   int64              `json:"duration_sec,omitempty"`
	LimitValue    int64              `json:"limit_value"`
	UsedValue     int64              `json:"used_value"`
	AllowedModels []string           `json:"allowed_models,omitempty"`
	WindowStart   string             `json:"window_start,omitempty"`
	WindowEnd     string             `json:"window_end,omitempty"`
	ResetAt       string             `json:"reset_at,omitempty"`
	Exceeded      bool               `json:"exceeded"`
}

// QuotaUpdate is the request payload to configure a key's quota rule.
type QuotaUpdate struct {
	Key           string             `json:"key,omitempty"`  // Can pass key or hash
	Hash          string             `json:"hash,omitempty"` // API key hash
	Enabled       bool               `json:"enabled"`
	Strategy      quota.StrategyType `json:"strategy"`
	Window        quota.WindowType   `json:"window"`
	DurationSec   int64              `json:"duration_sec,omitempty"`
	LimitValue    int64              `json:"limit_value"`
	AllowedModels []string           `json:"allowed_models,omitempty"`
}

// UsageSummary is the token accounting of one key inside the queried window.
type UsageSummary struct {
	usage.Counters
	Models []ModelUsage `json:"models,omitempty"`
}

// ModelUsage is the token accounting for one model of one key inside the
// queried window.
type ModelUsage struct {
	Model string `json:"model"`
	usage.Counters
}

// ResolveRequest carries the current proxy key list, in display order, and the
// window the usage figures must cover. Both bounds are RFC3339 instants; an
// empty bound means "the retained window".
type ResolveRequest struct {
	Keys  []string `json:"keys"`
	Start string   `json:"start,omitempty"`
	End   string   `json:"end,omitempty"`
}

// ResolveResult is the resolved metadata for the submitted key list.
type ResolveResult struct {
	Items []KeyEntry `json:"items"`
	// StaleHashes lists stored entries - remarks or usage counters - whose key is
	// absent from the submitted list. The page reports them back through Forget
	// once it has seen them stay stale for two consecutive reads.
	StaleHashes []string `json:"stale_hashes"`
}

// ForgetRequest carries the hash indexes whose plugin-side metadata - remark and
// usage counters - must be dropped.
type ForgetRequest struct {
	Hashes []string `json:"hashes"`
}

// ForgetResult reports how many keys actually had metadata to drop.
type ForgetResult struct {
	Status  string `json:"status"`
	Dropped int    `json:"dropped"`
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
	UpdateSettings(ctx context.Context, update SettingsUpdate) error
	Resolve(ctx context.Context, req ResolveRequest) (ResolveResult, error)
	Forget(ctx context.Context, hashes []string) (int, error)
	SetRemark(ctx context.Context, apiKey string, remark string) error
	GenerateKey(ctx context.Context) (string, error)
	Failures(ctx context.Context, req FailuresRequest) (FailuresResult, error)
	Quota(ctx context.Context, hash string) (QuotaSummary, bool, error)
	SetQuota(ctx context.Context, update QuotaUpdate) error
	DeleteQuota(ctx context.Context, hash string) error
	Models(ctx context.Context) ([]string, error)
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
	case PathPage:
		h.servePage(w, r)
	case PathResolve:
		h.serveResolve(w, r)
	case PathRemarks:
		h.serveRemarks(w, r)
	case PathForget:
		h.serveForget(w, r)
	case PathGenerate:
		h.serveGenerate(w, r)
	case PathFailures:
		h.serveFailures(w, r)
	case PathQuota:
		h.serveQuota(w, r)
	case PathModels:
		h.serveModels(w, r)
	default:
		writeJSON(w, http.StatusNotFound, errorBody("not_found", "unknown plugin route"))
	}
}

func (h *Handler) servePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(PageHTML))
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
		if update.UsageEnabled == nil && update.RetentionDays == nil {
			writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "usage_enabled or retention_days is required"))
			return
		}
		if h.fail(w, h.backend.UpdateSettings(r.Context(), update)) {
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
	result, err := h.backend.Resolve(r.Context(), request)
	if h.fail(w, err) {
		return
	}
	if result.Items == nil {
		result.Items = []KeyEntry{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) serveFailures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request FailuresRequest
	if !decodeBody(w, r, &request) {
		return
	}
	result, err := h.backend.Failures(r.Context(), request)
	if h.fail(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// serveForget drops the plugin-side metadata of the submitted keys. It is the
// only route that deletes stored data, and it is always an explicit write: a
// key list read is never destructive.
func (h *Handler) serveForget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request ForgetRequest
	if !decodeBody(w, r, &request) {
		return
	}
	if len(request.Hashes) == 0 {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "hashes are required"))
		return
	}
	dropped, err := h.backend.Forget(r.Context(), request.Hashes)
	if h.fail(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, ForgetResult{Status: "ok", Dropped: dropped})
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
func (h *Handler) serveQuota(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		hash := strings.TrimSpace(r.URL.Query().Get("hash"))
		if hash == "" {
			if k := strings.TrimSpace(r.URL.Query().Get("key")); k != "" {
				hash = remarks.Index(k)
			}
		}
		if hash == "" {
			writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "hash or key is required"))
			return
		}
		summary, found, err := h.backend.Quota(r.Context(), hash)
		if h.fail(w, err) {
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, errorBody("not_found", "quota rule not found"))
			return
		}
		writeJSON(w, http.StatusOK, summary)
	case http.MethodPut:
		var update QuotaUpdate
		if !decodeBody(w, r, &update) {
			return
		}
		if h.fail(w, h.backend.SetQuota(r.Context(), update)) {
			return
		}
		writeJSON(w, http.StatusOK, statusBody())
	case http.MethodDelete:
		hash := strings.TrimSpace(r.URL.Query().Get("hash"))
		if hash == "" {
			if k := strings.TrimSpace(r.URL.Query().Get("key")); k != "" {
				hash = remarks.Index(k)
			}
		}
		if hash == "" {
			writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "hash or key is required"))
			return
		}
		if h.fail(w, h.backend.DeleteQuota(r.Context(), hash)) {
			return
		}
		writeJSON(w, http.StatusOK, statusBody())
	default:
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPut+", "+http.MethodDelete)
	}
}
func (h *Handler) serveModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	models, err := h.backend.Models(r.Context())
	if h.fail(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"models": models})
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
