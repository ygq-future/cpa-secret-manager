package quota

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cpa-secret-manager/internal/usage"
)

// Manager coordinates quota rules, window states, and strategy evaluations.
// It is concurrent-safe and designed with zero host dependencies.
type Manager struct {
	mu          sync.RWMutex
	strategies  map[StrategyType]Strategy
	rules       map[string]Rule
	states      map[string]WindowState
	modelPrices map[string]ModelPrice
}

// NewManager initializes a quota manager with standard strategies.
func NewManager() *Manager {
	m := &Manager{
		strategies:  make(map[StrategyType]Strategy),
		rules:       make(map[string]Rule),
		states:      make(map[string]WindowState),
		modelPrices: make(map[string]ModelPrice),
	}
	m.RegisterStrategy(NewTokenLimitStrategy())
	return m
}

// SetModelPrices updates the model pricing table (3-tier prices per 1M tokens).
func (m *Manager) SetModelPrices(prices map[string]ModelPrice) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelPrices = make(map[string]ModelPrice, len(prices))
	for k, v := range prices {
		m.modelPrices[strings.ToLower(strings.TrimSpace(k))] = v
	}
}

// ModelPrices returns a copy of the active model pricing table.
func (m *Manager) ModelPrices() map[string]ModelPrice {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make(map[string]ModelPrice, len(m.modelPrices))
	for k, v := range m.modelPrices {
		res[k] = v
	}
	return res
}

// RegisterStrategy adds or replaces an evaluation strategy.
func (m *Manager) RegisterStrategy(s Strategy) {
	if s == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategies[s.Type()] = s
}

// GetRule returns the rule associated with the given key hash.
func (m *Manager) GetRule(hash string) (Rule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rule, ok := m.rules[hash]
	return rule, ok
}

// SetRule creates or updates the rule for a key hash.
func (m *Manager) SetRule(rule Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = time.Now().UTC()
	}
	m.rules[rule.Hash] = rule
}

// DeleteRule removes the rule for a key hash.
func (m *Manager) DeleteRule(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, hash)
}

// AllRules returns a shallow clone of all stored rules.
func (m *Manager) AllRules() map[string]Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make(map[string]Rule, len(m.rules))
	for k, v := range m.rules {
		res[k] = v
	}
	return res
}

// GetState returns the current window state for a key hash.
func (m *Manager) GetState(hash string) (WindowState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.states[hash]
	return state, ok
}

// AllStates returns a shallow clone of all window states.
func (m *Manager) AllStates() map[string]WindowState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make(map[string]WindowState, len(m.states))
	for k, v := range m.states {
		res[k] = v
	}
	return res
}

// Import restores rules and window states from persistent store.
func (m *Manager) Import(rules map[string]Rule, states map[string]WindowState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make(map[string]Rule, len(rules))
	for k, v := range rules {
		m.rules[k] = v
	}
	m.states = make(map[string]WindowState, len(states))
	for k, v := range states {
		m.states[k] = v
	}
}

// Forget removes rules and states for the given key hashes.
func (m *Manager) Forget(hashes []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, h := range hashes {
		delete(m.rules, h)
		delete(m.states, h)
	}
}

// Check evaluates whether a request for the given key hash is allowed under its active rule.
func (m *Manager) Check(ctx context.Context, hash string, model string, now time.Time) Decision {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, hasRule := m.rules[hash]
	if !hasRule || !rule.Enabled {
		return Decision{Allowed: true}
	}

	// Cost and dual strategies require the requested model to have a configured price.
	if rule.Strategy == StrategyTypeCost || rule.Strategy == StrategyTypeDual {
		trimmed := strings.ToLower(strings.TrimSpace(model))
		if trimmed == "" {
			return Decision{
				Allowed: false,
				Reason:  "model is required for cost-limited key",
			}
		}
		if _, hasPrice := m.modelPrices[trimmed]; !hasPrice {
			return Decision{
				Allowed: false,
				Reason:  fmt.Sprintf("model %q has no configured price for cost-limited key", model),
			}
		}
	}

	if len(rule.AllowedModels) > 0 && !isModelPermitted(rule.AllowedModels, model) {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("model %q is not permitted for this API key", model),
		}
	}

	wStart, wEnd := CalculateWindow(rule.Window, rule.DurationSec, now)
	state, hasState := m.states[hash]
	if !hasState || !state.WindowStart.Equal(wStart) || !state.WindowEnd.Equal(wEnd) {
		state = WindowState{
			Hash:        hash,
			WindowStart: wStart,
			WindowEnd:   wEnd,
			UsedValue:   0,
			UsedCost:    0,
		}
		m.states[hash] = state
	}

	// Token limit check (for token and dual strategies)
	if rule.Strategy == StrategyTypeToken || rule.Strategy == StrategyTypeDual {
		if rule.LimitValue > 0 && state.UsedValue >= rule.LimitValue {
			return Decision{
				Allowed:    false,
				Reason:     fmt.Sprintf("token limit exceeded: %d/%d tokens used in current window", state.UsedValue, rule.LimitValue),
				UsedValue:  state.UsedValue,
				LimitValue: rule.LimitValue,
				UsedCost:   state.UsedCost,
				LimitCost:  rule.LimitCost,
				ResetAt:    state.WindowEnd,
			}
		}
	}

	// Cost limit check (for cost and dual strategies)
	if rule.Strategy == StrategyTypeCost || rule.Strategy == StrategyTypeDual {
		if rule.LimitCost > 0 && state.UsedCost >= rule.LimitCost {
			return Decision{
				Allowed:    false,
				Reason:     fmt.Sprintf("cost limit exceeded: %.4f/%.4f used in current window", state.UsedCost, rule.LimitCost),
				UsedValue:  state.UsedValue,
				LimitValue: rule.LimitValue,
				UsedCost:   state.UsedCost,
				LimitCost:  rule.LimitCost,
				ResetAt:    state.WindowEnd,
			}
		}
	}

	// Fallback to registered strategy if non-standard
	if rule.Strategy != StrategyTypeToken && rule.Strategy != StrategyTypeCost && rule.Strategy != StrategyTypeDual {
		if strategy, hasStrat := m.strategies[rule.Strategy]; hasStrat {
			return strategy.Evaluate(ctx, rule, state, now)
		}
	}

	return Decision{
		Allowed:    true,
		UsedValue:  state.UsedValue,
		LimitValue: rule.LimitValue,
		UsedCost:   state.UsedCost,
		LimitCost:  rule.LimitCost,
		ResetAt:    state.WindowEnd,
	}
}

// Record accounts one request's usage towards the active quota window.
func (m *Manager) Record(record usage.Record) {
	if record.Hash == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rule, hasRule := m.rules[record.Hash]
	if !hasRule || !rule.Enabled {
		return
	}

	at := record.At
	if at.IsZero() {
		at = time.Now().UTC()
	}

	wStart, wEnd := CalculateWindow(rule.Window, rule.DurationSec, at)
	state, hasState := m.states[record.Hash]
	if !hasState || !state.WindowStart.Equal(wStart) || !state.WindowEnd.Equal(wEnd) {
		state = WindowState{
			Hash:        record.Hash,
			WindowStart: wStart,
			WindowEnd:   wEnd,
			UsedValue:   0,
			UsedCost:    0,
		}
	}

	tokens := record.Detail.Total
	if tokens == 0 && (record.Detail.Input > 0 || record.Detail.Output > 0) {
		tokens = record.Detail.Input + record.Detail.Output
	}
	state.UsedValue += tokens

	if p, ok := m.modelPrices[strings.ToLower(strings.TrimSpace(record.Model))]; ok {
		cacheTokens := record.Detail.CacheRead + record.Detail.Cached
		uncachedInput := record.Detail.Input
		totalComponents := record.Detail.Input + record.Detail.Output + record.Detail.Reasoning + cacheTokens
		if record.Detail.Total == 0 || record.Detail.Total < totalComponents {
			if uncachedInput >= cacheTokens {
				uncachedInput -= cacheTokens
			} else {
				uncachedInput = 0
			}
		}
		inTokens := uncachedInput + record.Detail.CacheCreation
		outTokens := record.Detail.Output + record.Detail.Reasoning
		if outTokens == 0 && tokens > 0 && inTokens == 0 && cacheTokens == 0 {
			outTokens = tokens
		}
		cachePrice := p.Cache
		if cachePrice == 0 && p.Input > 0 {
			cachePrice = p.Input
		}
		cost := (float64(inTokens)*p.Input + float64(outTokens)*p.Output + float64(cacheTokens)*cachePrice) / 1000000.0
		state.UsedCost += cost
	}
	state.RequestCount++
	state.LastAccess = at
	m.states[record.Hash] = state
}
func isModelPermitted(allowed []string, model string) bool {
	if len(allowed) == 0 {
		return true
	}
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return true
	}
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimSpace(a), trimmed) {
			return true
		}
	}
	return false
}
