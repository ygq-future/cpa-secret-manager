// Package runtime owns the plugin lifecycle: host RPC dispatch, plugin state
// and the Management API surface. It is the only layer allowed to mutate
// persisted plugin state.
package runtime

import (
	"context"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cpa-secret-manager/internal/config"
	"cpa-secret-manager/internal/errorslog"
	"cpa-secret-manager/internal/keys"
	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/quota"
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
	quota         *quota.Manager
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
	bucketStore    *state.BucketStore
	errorsStore    *errorslog.Store
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
		quota:         quota.NewManager(),
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
	case MethodRequestInterceptBefore:
		return r.handleInterceptBefore(ctx, raw)
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

	settings := management.Settings{
		UsageEnabled:  true,
		RetentionDays: state.DefaultRetentionDays,
		UsageWarning:  r.usageWarning,
	}
	if r.store != nil {
		cfg := r.store.AppConfig()
		settings.UsageEnabled = cfg.UsageEnabled
		settings.RetentionDays = cfg.RetentionDays
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
func (r *Runtime) UpdateSettings(_ context.Context, update management.SettingsUpdate) error {
	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if closed || store == nil {
		return ErrShutdown
	}
	current := store.AppConfig()
	if update.UsageEnabled != nil {
		current.UsageEnabled = *update.UsageEnabled
	}
	if update.RetentionDays != nil {
		current.RetentionDays = *update.RetentionDays
	}
	store.SetAppConfig(current)
	// Persisting right away also applies the new retention window: shrinking it
	// must drop the now-expired buckets immediately instead of waiting for the
	// next request to dirty the aggregate.
	return r.persistStateLocked(store)
}

// Failures implements management.Backend.
func (r *Runtime) Failures(_ context.Context, req management.FailuresRequest) (management.FailuresResult, error) {
	r.mu.RLock()
	errorsStore := r.errorsStore
	closed := r.closed
	r.mu.RUnlock()
	if closed || errorsStore == nil {
		return management.FailuresResult{}, nil
	}

	start, _ := parseInstant(req.Start)
	end, _ := parseInstant(req.End)

	if strings.TrimSpace(req.Hash) != "" {
		return management.FailuresResult{
			Items: errorsStore.Query(req.Hash, start, end),
		}, nil
	}

	all := errorsStore.QueryAll(start, end)
	var total []errorslog.ErrorEntry
	for _, list := range all {
		total = append(total, list...)
	}
	sort.Slice(total, func(i, j int) bool {
		return total[i].Time.After(total[j].Time)
	})
	return management.FailuresResult{Items: total}, nil
}

// Resolve implements management.Backend: it maps the submitted proxy key list
// onto stored metadata and usage, preserving list order. It is a pure read -
// deleting plugin state is a separate, explicit call (Forget), so a mistimed or
// mistrusted key list can never destroy anything.
func (r *Runtime) Resolve(_ context.Context, req management.ResolveRequest) (management.ResolveResult, error) {
	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if closed || store == nil {
		return management.ResolveResult{}, ErrShutdown
	}

	start, end := r.usageWindow(req)
	var aggregate map[string]usage.KeyUsage
	if end.After(start) {
		aggregate = r.aggregator.Range(start, end)
	}
	lastSeen := r.aggregator.LastSeen()

	items := make([]management.KeyEntry, 0, len(req.Keys))
	live := make(map[string]struct{}, len(req.Keys))
	for _, apiKey := range req.Keys {
		hash := remarks.Index(apiKey)
		live[hash] = struct{}{}
		entry, _ := store.Remark(hash)

		item := management.KeyEntry{Hash: hash, Remark: entry.Remark, LastSeen: formatTimestamp(lastSeen[hash])}
		if usageEntry, ok := aggregate[hash]; ok && !usageEntry.IsZero() {
			item.Usage = buildUsageSummary(usageEntry)
		}
		if rule, hasRule := r.quota.GetRule(hash); hasRule {
			item.Quota = r.buildQuotaSummary(hash, rule)
		}
		items = append(items, item)
	}

	return management.ResolveResult{Items: items, StaleHashes: r.staleHashes(store, live)}, nil
}

// usageWindow clamps a requested range to the retention window. A missing bound
// falls back to the retained window, so a caller that sends no range still sees
// exactly the history the plugin is allowed to keep. A range that falls outside
// retention collapses to an empty window, never to "everything".
func (r *Runtime) usageWindow(req management.ResolveRequest) (time.Time, time.Time) {
	now := r.clock.Now().UTC()
	start := now.AddDate(0, 0, -r.retentionDays())
	end := now
	if parsed, ok := parseInstant(req.Start); ok {
		start = parsed
	}
	if parsed, ok := parseInstant(req.End); ok {
		end = parsed
	}
	if earliest := now.AddDate(0, 0, -r.retentionDays()); start.Before(earliest) {
		start = earliest
	}
	if end.After(now) {
		end = now
	}
	return start, end
}

// retentionDays returns the configured retention with the canonical fallback.
func (r *Runtime) retentionDays() int {
	days := state.DefaultRetentionDays
	r.mu.RLock()
	if r.store != nil {
		days = r.store.AppConfig().RetentionDays
	}
	r.mu.RUnlock()
	if days < state.MinRetentionDays {
		days = state.DefaultRetentionDays
	}
	return days
}

func (r *Runtime) staleHashes(store *state.Store, live map[string]struct{}) []string {
	stale := make([]string, 0)
	for hash := range store.Remarks() {
		if _, ok := live[hash]; !ok {
			stale = append(stale, hash)
		}
	}
	for hash := range r.aggregator.Hashes() {
		if _, ok := live[hash]; !ok {
			stale = append(stale, hash)
		}
	}
	for hash := range r.quota.AllRules() {
		if _, ok := live[hash]; !ok {
			stale = append(stale, hash)
		}
	}
	sort.Strings(stale)
	return dedupeHashes(stale)
}

func dedupeHashes(hashes []string) []string {
	if len(hashes) < 2 {
		return hashes
	}
	out := hashes[:1]
	for _, hash := range hashes[1:] {
		if hash != out[len(out)-1] {
			out = append(out, hash)
		}
	}
	return out
}

// Forget implements management.Backend: it drops the plugin-side metadata of the
// submitted hash indexes - remark and usage counters alike. The page calls it
// when the operator deletes a key here, or after it confirmed that the key is
// gone from the host's list. Because the call is destructive, the state document
// is copied to <state>.bak first.
func (r *Runtime) Forget(_ context.Context, hashes []string) (int, error) {
	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	store := r.store
	closed := r.closed
	r.mu.RUnlock()
	if closed || store == nil {
		return 0, ErrShutdown
	}

	aggregate := r.aggregator.Hashes()
	pending := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		_, hasRemark := store.Remark(hash)
		_, hasUsage := aggregate[hash]
		_, hasRule := r.quota.GetRule(hash)
		_, hasState := r.quota.GetState(hash)
		if hasRemark || hasUsage || hasRule || hasState {
			pending = append(pending, hash)
		}
	}
	if len(pending) == 0 {
		return 0, nil
	}

	// Materialize the live buckets first: the recovery copy must contain them,
	// not just whatever the last flush wrote.
	r.mu.RLock()
	bucketStore := r.bucketStore
	r.mu.RUnlock()
	if bucketStore != nil {
		bucketStore.ReplaceBuckets(r.aggregator.SnapshotBuckets())
		if err := bucketStore.Backup(); err != nil {
			return 0, err
		}
	}
	if err := store.Backup(); err != nil {
		return 0, err
	}
	for _, hash := range pending {
		store.DeleteRemark(hash)
		r.aggregator.Remove(hash)
		if r.errorsStore != nil {
			r.errorsStore.Forget(hash)
		}
	}
	r.quota.Forget(pending)
	if err := r.persistStateLocked(store); err != nil {
		return 0, err
	}
	return len(pending), nil
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

// Quota implements management.Backend.
func (r *Runtime) Quota(_ context.Context, hash string) (management.QuotaSummary, bool, error) {
	r.mu.RLock()
	closed := r.closed
	store := r.store
	r.mu.RUnlock()
	if closed || store == nil {
		return management.QuotaSummary{}, false, ErrShutdown
	}

	rule, ok := r.quota.GetRule(hash)
	if !ok {
		return management.QuotaSummary{}, false, nil
	}

	summary := r.buildQuotaSummary(hash, rule)
	if summary == nil {
		return management.QuotaSummary{}, false, nil
	}
	return *summary, true, nil
}

// SetQuota implements management.Backend.
func (r *Runtime) SetQuota(_ context.Context, update management.QuotaUpdate) error {
	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	closed := r.closed
	store := r.store
	r.mu.RUnlock()
	if closed || store == nil {
		return ErrShutdown
	}

	hash := strings.TrimSpace(update.Hash)
	if hash == "" && update.Key != "" {
		if err := remarks.ValidateKey(update.Key); err != nil {
			return management.NewInvalidRequest(err.Error())
		}
		hash = remarks.Index(update.Key)
	}
	if hash == "" {
		return management.NewInvalidRequest("key or hash is required")
	}

	if update.LimitValue < 0 {
		return management.NewInvalidRequest("limit_value cannot be negative")
	}
	strategy := update.Strategy
	if strategy == "" {
		strategy = quota.StrategyTypeToken
	}
	window := update.Window
	if window == "" {
		window = quota.WindowTypeDaily
	}

	rule := quota.Rule{
		Hash:          hash,
		Enabled:       update.Enabled,
		Strategy:      strategy,
		Window:        window,
		DurationSec:   update.DurationSec,
		LimitValue:    update.LimitValue,
		AllowedModels: update.AllowedModels,
		UpdatedAt:     r.clock.Now().UTC(),
	}

	r.quota.SetRule(rule)
	return r.persistStateLocked(store)
}

// DeleteQuota implements management.Backend.
func (r *Runtime) DeleteQuota(_ context.Context, hash string) error {
	r.runMu.Lock()
	defer r.runMu.Unlock()

	r.mu.RLock()
	closed := r.closed
	store := r.store
	r.mu.RUnlock()
	if closed || store == nil {
		return ErrShutdown
	}

	r.quota.DeleteRule(hash)
	return r.persistStateLocked(store)
}

// Models implements management.Backend: returns all active models discovered by the plugin.
func (r *Runtime) Models(_ context.Context) ([]string, error) {
	r.mu.RLock()
	closed := r.closed
	store := r.store
	r.mu.RUnlock()
	if closed || store == nil {
		return nil, ErrShutdown
	}

	modelSet := r.aggregator.Models()
	if r.errorsStore != nil {
		for _, failures := range r.errorsStore.QueryAll(time.Time{}, time.Time{}) {
			for _, f := range failures {
				if f.Model != "" {
					modelSet[f.Model] = struct{}{}
				}
			}
		}
	}

	out := make([]string, 0, len(modelSet))
	for m := range modelSet {
		out = append(out, m)
	}
	sort.Strings(out)
	return out, nil
}

func (r *Runtime) buildQuotaSummary(hash string, rule quota.Rule) *management.QuotaSummary {
	now := r.clock.Now().UTC()
	wStart, wEnd := quota.CalculateWindow(rule.Window, rule.DurationSec, now)
	state, hasState := r.quota.GetState(hash)
	var used int64
	if hasState && state.WindowStart.Equal(wStart) && state.WindowEnd.Equal(wEnd) {
		used = state.UsedValue
	}

	exceeded := rule.Enabled && rule.LimitValue > 0 && used >= rule.LimitValue

	return &management.QuotaSummary{
		Enabled:       rule.Enabled,
		Strategy:      rule.Strategy,
		Window:        rule.Window,
		DurationSec:   rule.DurationSec,
		LimitValue:    rule.LimitValue,
		UsedValue:     used,
		AllowedModels: rule.AllowedModels,
		WindowStart:   formatTimestamp(wStart),
		WindowEnd:     formatTimestamp(wEnd),
		ResetAt:       formatTimestamp(wEnd),
		Exceeded:      exceeded,
	}
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
		// The record carries no managed key, so there is nothing to attribute it
		// to and nothing to count.
		return
	}
	if record.At.IsZero() {
		record.At = r.clock.Now().UTC()
	}
	r.aggregator.Observe(record)
	r.quota.Record(record)
	if r.errorsStore != nil && record.Failed && record.Hash != "" {
		r.errorsStore.Record(record.Hash, errorslog.ErrorEntry{
			Time:       record.At,
			Model:      record.Model,
			StatusCode: record.FailureStatus,
			Message:    record.FailureMessage,
		})
	}
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
		r.quota.Import(store.QuotaRules(), store.QuotaStates())
	}
	var errorsStore *errorslog.Store
	var bucketStore *state.BucketStore
	if strings.TrimSpace(path) != "" {
		errorsStore, _ = errorslog.Load(deriveErrorsPath(path))
		bucketStore, _ = state.LoadBuckets(deriveSiblingPath(path, "-buckets"))
	}
	if bucketStore != nil {
		r.aggregator.SeedBuckets(bucketStore.Buckets())
	}
	r.mu.Lock()
	r.store = store
	r.bucketStore = bucketStore
	r.errorsStore = errorsStore
	r.stateErr = err
	r.mu.Unlock()
}

// ErrorsPath returns the error log document that belongs to a plugin state
// path. The error log lives in its own file, so anything that resets the plugin
// state (local harnesses) must clear both.
func ErrorsPath(statePath string) string {
	return deriveErrorsPath(statePath)
}

// BucketsPath returns the time-bucketed usage document that belongs to a plugin
// state path.
func BucketsPath(statePath string) string {
	return deriveSiblingPath(statePath, "-buckets")
}

func deriveErrorsPath(statePath string) string {
	return deriveSiblingPath(statePath, "-errors")
}

// deriveSiblingPath names a document that sits beside the state file, keeping
// its directory and extension so one configured state path maps to exactly one
// set of documents.
func deriveSiblingPath(statePath, suffix string) string {
	trimmed := strings.TrimSpace(statePath)
	if trimmed == "" {
		return ""
	}
	ext := filepath.Ext(trimmed)
	if ext != "" {
		return strings.TrimSuffix(trimmed, ext) + suffix + ext
	}
	return trimmed + suffix + ".json"
}

// parseInstant parses an RFC3339 instant. An absent or malformed value reports
// false so callers can keep their own default instead of failing the request.
func parseInstant(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func (r *Runtime) registrationResult() RegisterResult {
	return RegisterResult{
		SchemaVersion: SchemaVersion,
		Metadata:      buildMetadata(),
		Capabilities:  Capabilities{ManagementAPI: true, UsagePlugin: true, RequestInterceptor: true},
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

	return r.persistStateLocked(store)
}

// persistStateLocked materializes the business settings and the usage buckets
// and writes both documents atomically. Callers must hold runMu.
func (r *Runtime) persistStateLocked(store *state.Store) error {
	r.mu.RLock()
	bucketStore := r.bucketStore
	errorsStore := r.errorsStore
	r.mu.RUnlock()

	store.ReplaceQuotaRules(r.quota.AllRules())
	store.ReplaceQuotaStates(r.quota.AllStates())
	if err := store.SaveAtomic(); err != nil {
		r.setUsageWarning(err.Error())
		return err
	}

	retentionDays := store.AppConfig().RetentionDays
	if retentionDays < state.MinRetentionDays {
		retentionDays = state.DefaultRetentionDays
	}
	cutoff := r.clock.Now().UTC().AddDate(0, 0, -retentionDays)

	if errorsStore != nil {
		errorsStore.Prune(cutoff)
		if errorsStore.Dirty() {
			_ = errorsStore.Save()
		}
	}
	// Prune before writing so a bucket that fell out of the retention window is
	// never written back.
	r.aggregator.Prune(cutoff)
	if bucketStore != nil {
		bucketStore.ReplaceBuckets(r.aggregator.SnapshotBuckets())
		if err := bucketStore.SaveAtomic(); err != nil {
			r.setUsageWarning(err.Error())
			return err
		}
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
	summary := &management.UsageSummary{Counters: entry.Counters}
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
