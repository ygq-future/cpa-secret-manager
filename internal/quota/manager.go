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
	mu         sync.RWMutex
	strategies map[StrategyType]Strategy
	rules      map[string]Rule
	states     map[string]WindowState
}

// NewManager initializes a quota manager with standard strategies.
func NewManager() *Manager {
	m := &Manager{
		strategies: make(map[StrategyType]Strategy),
		rules:      make(map[string]Rule),
		states:     make(map[string]WindowState),
	}
	m.RegisterStrategy(NewTokenLimitStrategy())
	return m
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

	if len(rule.AllowedModels) > 0 && !isModelPermitted(rule.AllowedModels, model) {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("model %q is not permitted for this API key", model),
		}
	}

	strategy, hasStrat := m.strategies[rule.Strategy]
	if !hasStrat {
		// If strategy is unknown, fail open to avoid accidental blackouts
		return Decision{Allowed: true}
	}

	wStart, wEnd := CalculateWindow(rule.Window, rule.DurationSec, now)
	state, hasState := m.states[hash]
	if !hasState || !state.WindowStart.Equal(wStart) || !state.WindowEnd.Equal(wEnd) {
		// New or rotated window
		state = WindowState{
			Hash:        hash,
			WindowStart: wStart,
			WindowEnd:   wEnd,
			UsedValue:   0,
		}
		m.states[hash] = state
	}

	return strategy.Evaluate(ctx, rule, state, now)
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

	strategy, hasStrat := m.strategies[rule.Strategy]
	if !hasStrat {
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
		}
	}

	strategy.Accumulate(&state, record)
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
