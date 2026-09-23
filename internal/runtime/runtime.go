// Package runtime owns the plugin lifecycle: host RPC dispatch, plugin state
// and the Management API surface. It is the only layer allowed to mutate
// persisted plugin state.
package runtime

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"cpa-secret-manager/internal/config"
	"cpa-secret-manager/internal/keys"
	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/remarks"
	"cpa-secret-manager/internal/state"
	"cpa-secret-manager/internal/usage"
)

// defaultFlushInterval bounds how much usage accounting a crash can lose.
const defaultFlushInterval = 5 * time.Second

// Runtime is the plugin application layer.
type Runtime struct {
	clock         Clock
	generator     *keys.Generator
	aggregator    *usage.Aggregator
	flushInterval time.Duration

	// lifecycleMu serializes register, reconfigure and shutdown.
	lifecycleMu sync.Mutex
	// runMu single-flights state-mutating operations.
	runMu sync.Mutex
	// flushMu guards the usage flush loop handle.
	flushMu   sync.Mutex
	flushStop chan struct{}

	mu             sync.RWMutex
	cfg            config.Config
	store          *state.Store
	stateErr       error
	configWarnings []string
	usageWarning   string
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
	generator := keys.NewGenerator()
	if opts.Random != nil {
		generator = keys.NewGeneratorWithSource(opts.Random)
	}
	flushInterval := opts.FlushInterval
	if flushInterval <= 0 {
		flushInterval = defaultFlushInterval
	}

	r := &Runtime{
		clock:         clock,
		cfg:           cfg,
		generator:     generator,
		aggregator:    usage.NewAggregator(),
		flushInterval: flushInterval,
	}
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
	case MethodUsageHandle:
		return r.handleUsage(raw)
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
	r.mu.Unlock()

	r.stopFlushLoop()
	err := r.flushUsage(true)

	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	return err
}

// ManagementHandler exposes the plugin HTTP surface to local harnesses.
func (r *Runtime) ManagementHandler() http.Handler {
	return r.handler
}

// Settings implements management.Backend.
func (r *Runtime) Settings(_ context.Context) (management.Settings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	settings := management.Settings{UsageEnabled: true, UsageWarning: r.usageWarning}
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
	if closed || store == nil {
		return ErrShutdown
	}
	store.SetAppConfig(state.AppConfig{UsageEnabled: usageEnabled})
	return store.SaveAtomic()
}

// Resolve implements management.Backend: it maps the submitted proxy key list
// onto stored metadata and usage, preserving list order.
func (r *Runtime) Resolve(_ context.Context, apiKeys []string) (management.ResolveResult, error) {
	store := r.Store()
	if store == nil {
		return management.ResolveResult{}, ErrShutdown
	}
	aggregate, unattributed := r.aggregator.Snapshot()

	items := make([]management.KeyEntry, 0, len(apiKeys))
	resolved := make(map[string]struct{}, len(apiKeys))
	for _, apiKey := range apiKeys {
		hash := remarks.Index(apiKey)
		resolved[hash] = struct{}{}
		entry, _ := store.Remark(hash)

		item := management.KeyEntry{Hash: hash, Remark: entry.Remark}
		if usageEntry, ok := aggregate[hash]; ok && !usageEntry.IsZero() {
			item.Usage = buildUsageSummary(usageEntry)
		}
		items = append(items, item)
	}

	orphanRemarks := 0
	for hash := range store.Remarks() {
		if _, ok := resolved[hash]; !ok {
			orphanRemarks++
		}
	}
	orphanUsage := 0
	for hash := range aggregate {
		if _, ok := resolved[hash]; !ok {
			orphanUsage++
		}
	}

	return management.ResolveResult{
		Items:                items,
		UnattributedRequests: unattributed,
		OrphanRemarks:        orphanRemarks,
		OrphanUsage:          orphanUsage,
	}, nil
}

// SetRemark implements management.Backend: it stores a remark for one proxy API
// key. An empty remark clears the stored text.
func (r *Runtime) SetRemark(_ context.Context, apiKey string, remark string) error {
	if err := remarks.ValidateKey(apiKey); err != nil {
		return management.NewInvalidRequest(err.Error())
	}
	normalized := remarks.Normalize(remark)
	if err := remarks.Validate(normalized); err != nil {
		return management.NewInvalidRequest(err.Error())
	}

	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if closed || store == nil {
		return ErrShutdown
	}

	hash := remarks.Index(apiKey)
	if normalized == "" {
		store.DeleteRemark(hash)
	} else {
		store.SetRemark(hash, remarks.Entry{Remark: normalized, UpdatedAt: r.clock.Now().UTC()})
	}
	return store.SaveAtomic()
}

// GenerateKey implements management.Backend: it returns one proxy API key in
// the official format.
func (r *Runtime) GenerateKey(_ context.Context) (string, error) {
	return r.generator.Generate()
}

// ObserveUsage accounts one usage record. It is exported so local harnesses can
// inject records without going through the host RPC path.
func (r *Runtime) ObserveUsage(record usage.Record) {
	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if closed || store == nil {
		return
	}
	if !store.AppConfig().UsageEnabled {
		return
	}
	if record.Hash == "" {
		r.aggregator.ObserveUnattributed()
		return
	}
	if record.At.IsZero() {
		record.At = r.clock.Now().UTC()
	}
	r.aggregator.Observe(record)
}

// FlushUsage persists pending usage counters when they changed since the last
// successful flush.
func (r *Runtime) FlushUsage() {
	_ = r.flushUsage(false)
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
	r.startFlushLoop()
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
	if store != nil {
		r.aggregator.Seed(store.Usage(), store.Metrics().UnattributedRequests)
	}
	r.mu.Lock()
	r.store = store
	r.stateErr = err
	r.mu.Unlock()
}

func (r *Runtime) registrationResult() RegisterResult {
	return RegisterResult{
		SchemaVersion: SchemaVersion,
		Metadata:      buildMetadata(),
		Capabilities:  Capabilities{ManagementAPI: true, UsagePlugin: true},
	}
}

func (r *Runtime) startFlushLoop() {
	r.flushMu.Lock()
	defer r.flushMu.Unlock()
	if r.flushStop != nil {
		return
	}
	stop := make(chan struct{})
	r.flushStop = stop
	go r.flushLoop(stop)
}

func (r *Runtime) flushLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			_ = r.flushUsage(false)
		}
	}
}

func (r *Runtime) stopFlushLoop() {
	r.flushMu.Lock()
	defer r.flushMu.Unlock()
	if r.flushStop == nil {
		return
	}
	close(r.flushStop)
	r.flushStop = nil
}

func (r *Runtime) flushUsage(force bool) error {
	if r.aggregator == nil {
		return nil
	}
	if !force && !r.aggregator.Dirty() {
		return nil
	}

	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if store == nil {
		return nil
	}
	if closed && !force {
		return nil
	}

	entries, unattributed := r.aggregator.Snapshot()
	store.ReplaceUsage(entries, unattributed)
	if err := store.SaveAtomic(); err != nil {
		r.setUsageWarning(err.Error())
		return err
	}
	r.aggregator.MarkClean()
	r.setUsageWarning("")
	return nil
}

func (r *Runtime) setUsageWarning(message string) {
	r.mu.Lock()
	r.usageWarning = message
	r.mu.Unlock()
}

func buildUsageSummary(entry usage.KeyUsage) *management.UsageSummary {
	summary := &management.UsageSummary{
		Counters:  entry.Counters,
		FirstSeen: formatTimestamp(entry.FirstSeen),
		LastSeen:  formatTimestamp(entry.LastSeen),
	}
	if len(entry.Models) == 0 {
		return summary
	}

	models := make([]management.ModelUsage, 0, len(entry.Models))
	for model, modelUsage := range entry.Models {
		models = append(models, management.ModelUsage{Model: model, Counters: modelUsage.Counters})
	}
	// Order by volume so the page renders a stable, useful list.
	sort.Slice(models, func(i, j int) bool {
		if models[i].Total != models[j].Total {
			return models[i].Total > models[j].Total
		}
		return models[i].Model < models[j].Model
	})
	summary.Models = models
	return summary
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
