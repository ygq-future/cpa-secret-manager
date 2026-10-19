package quota

import (
	"context"
	"time"

	"cpa-secret-manager/internal/usage"
)

// StrategyType identifies a quota evaluation strategy.
type StrategyType string

const (
	// StrategyTypeToken limits total tokens in a window.
	StrategyTypeToken StrategyType = "token"
	// StrategyTypeCost is reserved for future cost/price-based limit strategy.
	StrategyTypeCost StrategyType = "cost"
)

// WindowType identifies the duration or periodicity of a quota window.
type WindowType string

const (
	WindowTypeDaily    WindowType = "daily"    // Natural day (UTC)
	WindowTypeWeekly   WindowType = "weekly"   // Natural week (Monday 00:00 UTC)
	WindowTypeMonthly  WindowType = "monthly"  // Natural month (1st 00:00 UTC)
	WindowTypeDuration WindowType = "duration" // Fixed rolling/sliding duration
)

// Rule represents a quota limitation rule bound to an API key hash.
type Rule struct {
	Hash         string       `json:"hash"`
	Enabled      bool         `json:"enabled"`
	Strategy     StrategyType `json:"strategy"`
	Window       WindowType   `json:"window"`
	DurationSec  int64        `json:"duration_sec,omitempty"` // For WindowTypeDuration
	LimitValue    int64        `json:"limit_value"`            // e.g. total tokens
	AllowedModels []string     `json:"allowed_models,omitempty"`
	Remark        string       `json:"remark,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// WindowState tracks consumption within a specific time window.
type WindowState struct {
	Hash        string    `json:"hash"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	UsedValue   int64     `json:"used_value"`   // e.g. used tokens or micro-units
	RequestCount int64    `json:"request_count"`
	LastAccess  time.Time `json:"last_access"`
}

// Decision represents the result of a quota check.
type Decision struct {
	Allowed   bool
	Reason    string
	UsedValue int64
	LimitValue int64
	ResetAt   time.Time
}

// Strategy defines an abstraction for evaluating quota consumption and limits.
// Concrete implementations can be TokenLimitStrategy, CostLimitStrategy, etc.
type Strategy interface {
	Type() StrategyType
	// Evaluate assesses whether the request is allowed given the rule and window state.
	Evaluate(ctx context.Context, rule Rule, state WindowState, now time.Time) Decision
	// Accumulate updates the used value in the state from a usage record.
	Accumulate(state *WindowState, record usage.Record)
}
