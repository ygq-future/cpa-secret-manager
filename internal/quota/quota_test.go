package quota

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cpa-secret-manager/internal/usage"
)

func TestCalculateWindow(t *testing.T) {
	// Fixed reference time: 2026-10-19 14:30:00 UTC (Monday)
	ref := time.Date(2026, 10, 19, 14, 30, 0, 0, time.UTC)

	// Daily
	dStart, dEnd := CalculateWindow(WindowTypeDaily, 0, ref)
	wantDStart := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	wantDEnd := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if !dStart.Equal(wantDStart) || !dEnd.Equal(wantDEnd) {
		t.Fatalf("Daily window = [%v, %v), want [%v, %v)", dStart, dEnd, wantDStart, wantDEnd)
	}

	// Weekly (2026-10-19 is Monday)
	wStart, wEnd := CalculateWindow(WindowTypeWeekly, 0, ref)
	wantWStart := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	wantWEnd := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	if !wStart.Equal(wantWStart) || !wEnd.Equal(wantWEnd) {
		t.Fatalf("Weekly window = [%v, %v), want [%v, %v)", wStart, wEnd, wantWStart, wantWEnd)
	}

	// Monthly
	mStart, mEnd := CalculateWindow(WindowTypeMonthly, 0, ref)
	wantMStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	wantMEnd := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if !mStart.Equal(wantMStart) || !mEnd.Equal(wantMEnd) {
		t.Fatalf("Monthly window = [%v, %v), want [%v, %v)", mStart, mEnd, wantMStart, wantMEnd)
	}

	// Duration: 1800s (30m)
	durStart, durEnd := CalculateWindow(WindowTypeDuration, 1800, ref)
	wantDurStart := time.Date(2026, 10, 19, 14, 30, 0, 0, time.UTC)
	wantDurEnd := time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC)
	if !durStart.Equal(wantDurStart) || !durEnd.Equal(wantDurEnd) {
		t.Fatalf("Duration window = [%v, %v), want [%v, %v)", durStart, durEnd, wantDurStart, wantDurEnd)
	}
}

func TestTokenLimitStrategy_Evaluate(t *testing.T) {
	strategy := NewTokenLimitStrategy()
	rule := Rule{
		Hash:       "h1",
		Enabled:    true,
		Strategy:   StrategyTypeToken,
		Window:     WindowTypeDaily,
		LimitValue: 1000,
	}

	resetAt := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	state := WindowState{
		Hash:        "h1",
		WindowStart: time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		WindowEnd:   resetAt,
		UsedValue:   500,
	}

	// Case 1: Under limit -> allowed
	dec := strategy.Evaluate(context.Background(), rule, state, time.Now())
	if !dec.Allowed {
		t.Fatalf("expected allowed, got %+v", dec)
	}

	// Case 2: At or above limit -> rejected
	state.UsedValue = 1000
	dec = strategy.Evaluate(context.Background(), rule, state, time.Now())
	if dec.Allowed {
		t.Fatalf("expected rejected at limit, got %+v", dec)
	}
	if dec.ResetAt != resetAt {
		t.Fatalf("expected reset at %v, got %v", resetAt, dec.ResetAt)
	}

	// Case 3: LimitValue <= 0 -> unlimited, allowed
	rule.LimitValue = 0
	dec = strategy.Evaluate(context.Background(), rule, state, time.Now())
	if !dec.Allowed {
		t.Fatalf("expected allowed for unlimited rule, got %+v", dec)
	}
}

func TestManager_CheckAndRecord(t *testing.T) {
	mgr := NewManager()
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

	// Key without rule -> allowed
	if dec := mgr.Check(context.Background(), "h1", "gpt-5.6", now); !dec.Allowed {
		t.Fatalf("expected allowed without rule")
	}

	// Set rule for h1: limit 500 tokens / daily
	mgr.SetRule(Rule{
		Hash:       "h1",
		Enabled:    true,
		Strategy:   StrategyTypeToken,
		Window:     WindowTypeDaily,
		LimitValue: 500,
	})

	// Initial check -> allowed
	if dec := mgr.Check(context.Background(), "h1", "gpt-5.6", now); !dec.Allowed {
		t.Fatalf("expected allowed before usage")
	}

	// Record usage: 300 tokens
	mgr.Record(usage.Record{
		Hash: "h1",
		At:   now,
		Detail: usage.Detail{
			Total: 300,
		},
	})

	// Check again -> still allowed (300 < 500)
	dec := mgr.Check(context.Background(), "h1", "gpt-5.6", now)
	if !dec.Allowed {
		t.Fatalf("expected allowed at 300/500, got %+v", dec)
	}
	if dec.UsedValue != 300 {
		t.Fatalf("expected UsedValue = 300, got %d", dec.UsedValue)
	}

	// Record usage: 250 tokens -> total 550 tokens
	mgr.Record(usage.Record{
		Hash: "h1",
		At:   now.Add(time.Minute),
		Detail: usage.Detail{
			Total: 250,
		},
	})

	// Check again -> rejected (550 >= 500)
	dec = mgr.Check(context.Background(), "h1", "gpt-5.6", now.Add(2*time.Minute))
	if dec.Allowed {
		t.Fatalf("expected rejected at 550/500, got %+v", dec)
	}

	// Fast forward to next day -> window rotated, should be allowed
	nextDay := now.AddDate(0, 0, 1)
	decNext := mgr.Check(context.Background(), "h1", "gpt-5.6", nextDay)
	if !decNext.Allowed {
		t.Fatalf("expected allowed on next day window, got %+v", decNext)
	}
	if decNext.UsedValue != 0 {
		t.Fatalf("expected UsedValue = 0 on reset, got %d", decNext.UsedValue)
	}
}

func TestManager_ForgetAndImport(t *testing.T) {
	mgr := NewManager()
	mgr.SetRule(Rule{
		Hash:       "h1",
		Enabled:    true,
		Strategy:   StrategyTypeToken,
		Window:     WindowTypeDaily,
		LimitValue: 100,
	})
	mgr.Record(usage.Record{
		Hash: "h1",
		At:   time.Now().UTC(),
		Detail: usage.Detail{
			Total: 50,
		},
	})

	rules := mgr.AllRules()
	states := mgr.AllStates()
	if len(rules) != 1 || len(states) != 1 {
		t.Fatalf("expected 1 rule and 1 state")
	}

	// Create new manager and import
	mgr2 := NewManager()
	mgr2.Import(rules, states)
	if r, ok := mgr2.GetRule("h1"); !ok || r.LimitValue != 100 {
		t.Fatalf("imported rule mismatch: %+v", r)
	}
	if s, ok := mgr2.GetState("h1"); !ok || s.UsedValue != 50 {
		t.Fatalf("imported state mismatch: %+v", s)
	}

	// Forget h1
	mgr2.Forget([]string{"h1"})
	if _, ok := mgr2.GetRule("h1"); ok {
		t.Fatalf("expected rule to be removed")
	}
	if _, ok := mgr2.GetState("h1"); ok {
		t.Fatalf("expected state to be removed")
	}
}

func TestManager_Concurrency(t *testing.T) {
	mgr := NewManager()
	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		mgr.SetRule(Rule{
			Hash:       fmt.Sprintf("h%d", i),
			Enabled:    true,
			Strategy:   StrategyTypeToken,
			Window:     WindowTypeDaily,
			LimitValue: 10000,
		})
	}

	var wg sync.WaitGroup
	for worker := 0; worker < 20; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			hash := fmt.Sprintf("h%d", id%10)
			for j := 0; j < 50; j++ {
				mgr.Check(context.Background(), hash, "gpt-5.6", now)
				mgr.Record(usage.Record{
					Hash:   hash,
					At:     now,
					Detail: usage.Detail{Total: 10},
				})
			}
		}(worker)
	}
	wg.Wait()

	states := mgr.AllStates()
	for i := 0; i < 10; i++ {
		h := fmt.Sprintf("h%d", i)
		st, ok := states[h]
		if !ok || st.UsedValue != 1000 {
			t.Fatalf("expected UsedValue=1000 for %s, got %+v", h, st)
		}
	}
}

func TestManager_AllowedModels(t *testing.T) {
	mgr := NewManager()
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

	mgr.SetRule(Rule{
		Hash:          "h1",
		Enabled:       true,
		Strategy:      StrategyTypeToken,
		Window:        WindowTypeDaily,
		LimitValue:    5000,
		AllowedModels: []string{"gpt-5.6", "claude-3-7-sonnet"},
	})

	// 1. Whitelisted model -> allowed
	if dec := mgr.Check(context.Background(), "h1", "gpt-5.6", now); !dec.Allowed {
		t.Fatalf("expected gpt-5.6 allowed, got: %+v", dec)
	}
	if dec := mgr.Check(context.Background(), "h1", "CLAUDE-3-7-SONNET", now); !dec.Allowed {
		t.Fatalf("expected case-insensitive match allowed, got: %+v", dec)
	}

	// 2. Non-whitelisted model -> rejected
	dec := mgr.Check(context.Background(), "h1", "gemini-2.5-pro", now)
	if dec.Allowed {
		t.Fatal("expected gemini-2.5-pro rejected")
	}
	if dec.Reason != `model "gemini-2.5-pro" is not permitted for this API key` {
		t.Fatalf("unexpected reason: %q", dec.Reason)
	}
}
