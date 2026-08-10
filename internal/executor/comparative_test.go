package executor

import (
	"context"
	"sort"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// compareSuite builds a two-variant eval whose passing is decided by an
// output_contains step (no extra runner calls), plus a suite-level compare
// block. Both variants pass when their agent output contains "MARKER".
func compareSuite() *suite.Suite {
	return &suite.Suite{
		Description: "compare suite",
		Compare:     &suite.Compare{Criteria: []string{"clarity"}},
		Evals: []suite.Eval{{
			ID:     "eval-1",
			Name:   "Eval One",
			Prompt: "do something",
			Verify: []suite.VerifyStep{{Type: "output_contains", Values: []string{"MARKER"}}},
			Variants: []suite.Variant{
				{Name: "control", Runner: "claude-code"},
				{Name: "treatment", Runner: "claude-code"},
			},
		}},
	}
}

const twoScores = `{"scores":[{"label":"A","rating":4,"reason":"a"},{"label":"B","rating":2,"reason":"b"}]}`

func TestComparison_ScoresBothPassingVariants(t *testing.T) {
	runner := &fakeRunner{results: []*agentrunner.Result{
		{Text: "control MARKER"},
		{Text: "treatment MARKER"},
		{Text: twoScores},
	}}

	sr, err := Execute(context.Background(), compareSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := sr.Evals[0].Comparison
	if c == nil {
		t.Fatal("expected a comparison result")
	}
	if c.Skipped != "" {
		t.Fatalf("expected comparison to run, got skipped: %s", c.Skipped)
	}
	if len(c.Scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(c.Scores))
	}

	// Both variants scored; ratings are the set {4,2} (variant order depends on
	// the judge's shuffle, which is exercised deterministically in the verifier
	// unit tests).
	var ratings []int
	scoredNames := map[string]bool{}
	for _, s := range c.Scores {
		scoredNames[s.Variant] = true
		ratings = append(ratings, s.Rating)
	}
	if !scoredNames["control"] || !scoredNames["treatment"] {
		t.Errorf("both variants should be scored, got %v", scoredNames)
	}
	sort.Ints(ratings)
	if ratings[0] != 2 || ratings[1] != 4 {
		t.Errorf("expected ratings {2,4}, got %v", ratings)
	}
	if c.Model == "" {
		t.Error("expected judge model recorded on comparison")
	}
}

func TestComparison_SkippedWhenOnlyOnePasses(t *testing.T) {
	runner := &fakeRunner{results: []*agentrunner.Result{
		{Text: "control MARKER"},
		{Text: "treatment WITHOUT the token"}, // fails output_contains
	}}

	sr, err := Execute(context.Background(), compareSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := sr.Evals[0].Comparison
	if c == nil {
		t.Fatal("expected a comparison result recording the skip")
	}
	if c.Skipped == "" {
		t.Error("expected comparison to be skipped with only one passing variant")
	}
	if len(c.Scores) != 0 {
		t.Errorf("expected no scores, got %d", len(c.Scores))
	}
}

func TestComparison_DegradesOnUnparseableJudge(t *testing.T) {
	runner := &fakeRunner{results: []*agentrunner.Result{
		{Text: "control MARKER"},
		{Text: "treatment MARKER"},
		{Text: "I liked them all the same"}, // no JSON
	}}

	sr, err := Execute(context.Background(), compareSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := sr.Evals[0].Comparison
	if c == nil || c.Skipped == "" {
		t.Fatal("expected comparison to degrade gracefully with a skip reason")
	}
	if len(c.Scores) != 0 {
		t.Errorf("expected no scores on degrade, got %d", len(c.Scores))
	}
	// Per-run pass/fail must be untouched by the comparison failure.
	for _, v := range sr.Evals[0].Variants {
		if v.Runs[0].Pass == nil || !*v.Runs[0].Pass {
			t.Errorf("variant %q run should still be marked passing", v.Name)
		}
	}
}

func TestComparison_DisabledByDefault(t *testing.T) {
	s := compareSuite()
	s.Compare = nil // no comparison configured

	runner := &fakeRunner{results: []*agentrunner.Result{
		{Text: "control MARKER"},
		{Text: "treatment MARKER"},
	}}
	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sr.Evals[0].Comparison != nil {
		t.Error("expected no comparison when the suite does not configure it")
	}
}

func TestComparison_NoCompareOverride(t *testing.T) {
	runner := &fakeRunner{results: []*agentrunner.Result{
		{Text: "control MARKER"},
		{Text: "treatment MARKER"},
	}}
	off := false
	sr, err := Execute(context.Background(), compareSuite(), fakeRegistry(runner), &Options{Compare: &off})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sr.Evals[0].Comparison != nil {
		t.Error("--no-compare should disable comparison even when configured")
	}
}

func TestEligibleForComparison(t *testing.T) {
	variants := []result.VariantResult{
		{Name: "all-pass", Runs: []result.RunResult{
			{Text: "first", Pass: boolPtr(true)},
			{Text: "second", Pass: boolPtr(true)},
		}},
		{Name: "one-fail", Runs: []result.RunResult{
			{Text: "x", Pass: boolPtr(true)},
			{Text: "y", Pass: boolPtr(false)},
		}},
		{Name: "unverified", Runs: []result.RunResult{
			{Text: "z", Pass: nil},
		}},
	}
	got := eligibleForComparison(variants)
	if len(got) != 1 {
		t.Fatalf("expected 1 eligible variant, got %d", len(got))
	}
	if got[0].name != "all-pass" {
		t.Errorf("expected 'all-pass' eligible, got %q", got[0].name)
	}
	if got[0].output != "first" {
		t.Errorf("expected first passing run's output, got %q", got[0].output)
	}
}
