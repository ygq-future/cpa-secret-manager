// Package management owns the plugin HTTP surface: the plugin-owned Management
// API routes and the browser resource page. It depends on the runtime through
// the Backend interface only.
package management

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
	// RouteSettings is the route path registered with the host.
	RouteSettings = legacyBase + PathSettings
)

// Settings is the plugin business configuration exposed to the page.
type Settings struct {
	UsageEnabled   bool     `json:"usage_enabled"`
	StatePath      string   `json:"state_path"`
	StateWarning   string   `json:"state_warning,omitempty"`
	ConfigWarnings []string `json:"config_warnings,omitempty"`
}

// SettingsUpdate is the request payload for updating business settings.
type SettingsUpdate struct {
	UsageEnabled *bool `json:"usage_enabled"`
}

// Backend is the runtime surface the HTTP adapter is allowed to call.
type Backend interface {
	Settings(ctx context.Context) (Settings, error)
	UpdateSettings(ctx context.Context, usageEnabled bool) error
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

	switch normalizePath(r.URL.Path) {
	case PathSettings:
		h.serveSettings(w, r)
	default:
		writeJSON(w, http.StatusNotFound, errorBody("not_found", "unknown plugin route"))
	}
}

func (h *Handler) serveSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.backend == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorBody("backend_unavailable", "runtime not available"))
			return
		}
		settings, err := h.backend.Settings(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorBody("settings_failed", err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut, http.MethodPatch:
		if h.backend == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorBody("backend_unavailable", "runtime not available"))
			return
		}
		var update SettingsUpdate
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&update); err != nil {
			writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "request body must be a JSON object"))
			return
		}
		if update.UsageEnabled == nil {
			writeJSON(w, http.StatusBadRequest, errorBody("invalid_request", "usage_enabled is required"))
			return
		}
		if err := h.backend.UpdateSettings(r.Context(), *update.UsageEnabled); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorBody("settings_failed", err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, statusBody())
	default:
		w.Header().Set("Allow", "GET, PUT, PATCH")
		writeJSON(w, http.StatusMethodNotAllowed, errorBody("method_not_allowed", "method not allowed"))
	}
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
