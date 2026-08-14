package executor

import (
	"context"
	"testing"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/suite"
)

func TestBudget_NilIsInert(t *testing.T) {
	var b *budget
	b.add(1.0) // must not panic
	if b.stopped() {
		t.Error("nil budget should never report stopped")
	}
	if spent, exceeded := b.report(); spent != 0 || exceeded {
		t.Errorf("nil budget report should be zero/false, got %f/%v", spent, exceeded)
	}
}

func TestBudget_CrossesCap(t *testing.T) {
	b := newBudget(1.0)
	b.add(0.4)
	if b.stopped() {
		t.Error("budget should not be stopped below the cap")
	}
	b.add(0.7) // total 1.1 > 1.0
	if !b.stopped() {
		t.Error("budget should be stopped after crossing the cap")
	}
	spent, exceeded := b.report()
	if !exceeded || spent != 1.1 {
		t.Errorf("expected spent=1.1 exceeded=true, got %f/%v", spent, exceeded)
	}
}

func TestNewBudget_ZeroOrNegativeIsNil(t *testing.T) {
	if newBudget(0) != nil {
		t.Error("cap of 0 should yield a nil (inert) budget")
	}
	if newBudget(-5) != nil {
		t.Error("negative cap should yield a nil (inert) budget")
	}
}

// TestMaxCostAbortsRun verifies the suite-level circuit breaker: with a cap of
// $0.10 and samples costing $0.05 each, the run stops after cumulative spend
// crosses the cap rather than executing every planned sample.
func TestMaxCostAbortsRun(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "s1", CostUSD: 0.05},
			{Text: "s2", CostUSD: 0.05},
			{Text: "s3", CostUSD: 0.05}, // total 0.15 > 0.10: never reached
			{Text: "s4", CostUSD: 0.05},
			{Text: "s5", CostUSD: 0.05},
		},
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(5)

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{MaxCost: 0.10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	// Two samples take us to $0.10; the third crosses to $0.15 and flips the
	// cap, so exactly three samples run before the fourth is skipped.
	if len(runs) != 3 {
		t.Fatalf("expected 3 samples before abort, got %d", len(runs))
	}
	if sr.Abort == nil {
		t.Fatal("expected sr.Abort to be set")
	}
	if sr.Abort.CapUSD != 0.10 {
		t.Errorf("expected cap 0.10, got %f", sr.Abort.CapUSD)
	}
	if sr.Abort.SpentUSD < 0.149 || sr.Abort.SpentUSD > 0.151 {
		t.Errorf("expected spent ~0.15, got %f", sr.Abort.SpentUSD)
	}
}

// TestMaxCostStopsBeforeNextEval verifies the breaker prevents later evals from
// starting once the cap is crossed by an earlier eval.
func TestMaxCostStopsBeforeNextEval(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "e1", CostUSD: 0.20}, // crosses the 0.10 cap immediately
			{Text: "e2", CostUSD: 0.20},
		},
	}

	s := &suite.Suite{
		Description: "two evals",
		Evals: []suite.Eval{
			{ID: "eval-1", Name: "One", Prompt: "p", Variants: []suite.Variant{{Name: "control", Runner: "claude-code"}}},
			{ID: "eval-2", Name: "Two", Prompt: "p", Variants: []suite.Variant{{Name: "control", Runner: "claude-code"}}},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{MaxCost: 0.10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the first eval should have executed; the second is never started.
	if len(sr.Evals) != 1 {
		t.Fatalf("expected only 1 eval to run before abort, got %d", len(sr.Evals))
	}
	if sr.Abort == nil {
		t.Fatal("expected sr.Abort to be set")
	}
}

// TestNoMaxCostRunsEverything ensures a zero cap leaves behavior unchanged.
func TestNoMaxCostRunsEverything(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "s1", CostUSD: 1.0, Duration: time.Millisecond},
			{Text: "s2", CostUSD: 1.0, Duration: time.Millisecond},
			{Text: "s3", CostUSD: 1.0, Duration: time.Millisecond},
		},
	}
	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(3)

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sr.Evals[0].Variants[0].Runs) != 3 {
		t.Errorf("expected all 3 samples to run without a cap, got %d", len(sr.Evals[0].Variants[0].Runs))
	}
	if sr.Abort != nil {
		t.Errorf("expected no abort without a cap, got %+v", sr.Abort)
	}
}
