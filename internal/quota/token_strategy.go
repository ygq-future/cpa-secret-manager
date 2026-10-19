package quota

import (
	"context"
	"fmt"
	"time"

	"cpa-secret-manager/internal/usage"
)

// TokenLimitStrategy limits requests by total token consumption in the window.
type TokenLimitStrategy struct{}

// NewTokenLimitStrategy constructs a TokenLimitStrategy.
func NewTokenLimitStrategy() *TokenLimitStrategy {
	return &TokenLimitStrategy{}
}

func (s *TokenLimitStrategy) Type() StrategyType {
	return StrategyTypeToken
}

func (s *TokenLimitStrategy) Evaluate(_ context.Context, rule Rule, state WindowState, _ time.Time) Decision {
	if rule.LimitValue <= 0 {
		return Decision{
			Allowed:    true,
			UsedValue:  state.UsedValue,
			LimitValue: rule.LimitValue,
			ResetAt:    state.WindowEnd,
		}
	}

	if state.UsedValue >= rule.LimitValue {
		return Decision{
			Allowed:    false,
			Reason:     fmt.Sprintf("token limit exceeded: %d/%d tokens used in current window", state.UsedValue, rule.LimitValue),
			UsedValue:  state.UsedValue,
			LimitValue: rule.LimitValue,
			ResetAt:    state.WindowEnd,
		}
	}

	return Decision{
		Allowed:    true,
		UsedValue:  state.UsedValue,
		LimitValue: rule.LimitValue,
		ResetAt:    state.WindowEnd,
	}
}

func (s *TokenLimitStrategy) Accumulate(state *WindowState, record usage.Record) {
	if state == nil {
		return
	}
	tokens := record.Detail.Total
	if tokens == 0 && (record.Detail.Input > 0 || record.Detail.Output > 0) {
		tokens = record.Detail.Input + record.Detail.Output
	}
	state.UsedValue += tokens
	state.RequestCount++
	if !record.At.IsZero() {
		state.LastAccess = record.At
	} else {
		state.LastAccess = time.Now().UTC()
	}
}
