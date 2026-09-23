// Package runtime owns the plugin lifecycle: host RPC dispatch, plugin state
// and the Management API surface. It is the only layer allowed to mutate
// persisted plugin state.
package runtime

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"cpa-secret-manager/internal/config"
	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/state"
)

// Runtime is the plugin application layer.
type Runtime struct {
	clock Clock

	// lifecycleMu serializes register, reconfigure and shutdown.
	lifecycleMu sync.Mutex
	// runMu single-flights state-mutating operations.
	runMu sync.Mutex

	mu             sync.RWMutex
	cfg            config.Config
	store          *state.Store
	stateErr       error
	configWarnings []string
	closed         bool
	handler        *management.Handler
}

// New constructs a runtime with the given options.
func New(opts Options) *Runtime {
	clock := opts.Clock
	if clock == nil {
		clock = systemClock{}
	}
	cfg := config.Default()
	if trimmed := strings.TrimSpace(opts.StatePath); trimmed != "" {
		cfg.StatePath = trimmed
	}

	r := &Runtime{clock: clock, cfg: cfg}
	r.handler = management.NewHandler(r)
	r.loadState(cfg.StatePath)
	return r
}

// Handle dispatches one host RPC call and returns a JSON envelope response.
func (r *Runtime) Handle(ctx context.Context, method string, raw []byte) []byte {
	switch strings.TrimSpace(method) {
	case MethodPluginRegister, MethodPluginReconfigure:
		return r.handleRegister(raw)
	case MethodPluginShutdown:
		if err := r.Shutdown(ctx); err != nil {
			return failureEnvelope("shutdown_failed", err.Error())
		}
		return okEnvelope(map[string]string{"status": "ok"})
	case MethodManagementRegister:
		return r.handleManagementRegister()
	case MethodManagementHandle:
		return r.handleManagement(ctx, raw)
	default:
		return failureEnvelope("unknown_method", "unknown method: "+method)
	}
}

// Shutdown persists pending state and marks the runtime closed.
func (r *Runtime) Shutdown(_ context.Context) error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	store := r.store
	r.closed = true
	r.mu.Unlock()

	if store == nil {
		return nil
	}
	return store.SaveAtomic()
}

// ManagementHandler exposes the plugin HTTP surface to local harnesses.
func (r *Runtime) ManagementHandler() http.Handler {
	return r.handler
}

// Settings implements management.Backend.
func (r *Runtime) Settings(_ context.Context) (management.Settings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	settings := management.Settings{UsageEnabled: true}
	if r.store != nil {
		settings.UsageEnabled = r.store.AppConfig().UsageEnabled
		settings.StatePath = r.store.Path()
	}
	if len(r.configWarnings) > 0 {
		settings.ConfigWarnings = append([]string(nil), r.configWarnings...)
	}
	if r.stateErr != nil {
		settings.StateWarning = r.stateErr.Error()
	}
	return settings, nil
}

// UpdateSettings implements management.Backend.
func (r *Runtime) UpdateSettings(_ context.Context, usageEnabled bool) error {
	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if closed {
		return ErrShutdown
	}
	if store == nil {
		return ErrShutdown
	}
	store.SetAppConfig(state.AppConfig{UsageEnabled: usageEnabled})
	return store.SaveAtomic()
}

// Store exposes the plugin state store to local harnesses and tests.
func (r *Runtime) Store() *state.Store {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.store
}

func (r *Runtime) handleRegister(raw []byte) []byte {
	yamlText, err := decodeConfigYAML(raw)
	if err != nil {
		return failureEnvelope("invalid_request", err.Error())
	}
	cfg, warnings, err := config.LoadBytes([]byte(yamlText))
	if err != nil {
		return failureEnvelope("invalid_config", err.Error())
	}
	r.applyConfig(cfg, warnings)
	return okEnvelope(r.registrationResult())
}

func (r *Runtime) applyConfig(cfg config.Config, warnings []string) {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()

	r.mu.Lock()
	previous := r.cfg
	r.cfg = cfg
	r.configWarnings = warnings
	r.mu.Unlock()

	if previous.StatePath == cfg.StatePath {
		return
	}
	r.loadState(cfg.StatePath)
}

func (r *Runtime) loadState(path string) {
	store, err := state.Load(path)
	r.mu.Lock()
	r.store = store
	r.stateErr = err
	r.mu.Unlock()
}

func (r *Runtime) registrationResult() RegisterResult {
	return RegisterResult{
		SchemaVersion: SchemaVersion,
		Metadata:      buildMetadata(),
		Capabilities:  Capabilities{ManagementAPI: true},
	}
}
