package quota

import (
	"context"
	"encoding/json"
	"time"

	"cpa-secret-manager/internal/usage"
)

// StrategyType identifies a quota evaluation strategy.
type StrategyType string

const (
	// StrategyTypeToken limits total tokens in a window.
	StrategyTypeToken StrategyType = "token"
	// StrategyTypeCost limits total cost/price in a window.
	StrategyTypeCost StrategyType = "cost"
	// StrategyTypeDual limits both total tokens and cost in a window; either exceeded blocks.
	StrategyTypeDual StrategyType = "dual"
)

// ModelPrice specifies 3-tier pricing (input, output, cache) per million tokens.
type ModelPrice struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
	Cache  float64 `json:"cache"`
}

// UnmarshalJSON supports both legacy float64 numbers and new 3-tier pricing structs.
func (mp *ModelPrice) UnmarshalJSON(data []byte) error {
	var single float64
	if err := json.Unmarshal(data, &single); err == nil {
		mp.Input = single
		mp.Output = single
		mp.Cache = single
		return nil
	}
	type alias ModelPrice
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*mp = ModelPrice(a)
	return nil
}
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
	Hash          string       `json:"hash"`
	Enabled       bool         `json:"enabled"`
	Strategy      StrategyType `json:"strategy"`
	Window        WindowType   `json:"window"`
	DurationSec   int64        `json:"duration_sec,omitempty"` // For WindowTypeDuration
	LimitValue    int64        `json:"limit_value"`            // e.g. total tokens
	LimitCost     float64      `json:"limit_cost,omitempty"`   // e.g. total cost/price limit
	AllowedModels []string     `json:"allowed_models,omitempty"`
	Remark        string       `json:"remark,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// WindowState tracks consumption within a specific time window.
type WindowState struct {
	Hash         string    `json:"hash"`
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
	UsedValue    int64     `json:"used_value"`   // used tokens
	UsedCost     float64   `json:"used_cost"`    // used cost
	RequestCount int64     `json:"request_count"`
	LastAccess   time.Time `json:"last_access"`
}

// Decision represents the result of a quota check.
type Decision struct {
	Allowed    bool
	Reason     string
	UsedValue  int64
	LimitValue int64
	UsedCost   float64
	LimitCost  float64
	ResetAt    time.Time
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
