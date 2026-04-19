package compare

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

func float64Ptr(v float64) *float64 { return &v }

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func makeRuns(passResults []bool, cost float64, durationMs int64) []result.RunResult {
	runs := make([]result.RunResult, len(passResults))
	for i, p := range passResults {
		pass := p
		runs[i] = result.RunResult{
			Sample:     i + 1,
			Pass:       &pass,
			CostUSD:    cost,
			DurationMs: durationMs,
		}
	}
	return runs
}

func makeSuiteResult(desc string, evals []result.EvalResult) *result.SuiteResult {
	return &result.SuiteResult{
		Description: desc,
		StartedAt:   time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 4, 18, 10, 30, 0, 0, time.UTC),
		Evals:       evals,
	}
}

func TestCompare_MatchingVariants(t *testing.T) {
	baseline := makeSuiteResult("baseline", []result.EvalResult{
		{
			EvalID:   "eval-1",
			EvalName: "Fizzbuzz",
			Variants: []result.VariantResult{
				{Name: "control", Runs: makeRuns([]bool{true, true}, 0.05, 5000)},
				{Name: "variant", Runs: makeRuns([]bool{true, false}, 0.10, 8000)},
			},
		},
	})

	candidate := makeSuiteResult("candidate", []result.EvalResult{
		{
			EvalID:   "eval-1",
			EvalName: "Fizzbuzz",
			Variants: []result.VariantResult{
				{Name: "control", Runs: makeRuns([]bool{true, true}, 0.04, 4000)},
				{Name: "variant", Runs: makeRuns([]bool{true, true}, 0.08, 6000)},
			},
		},
	})

	c := Compare(baseline, candidate)

	if len(c.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(c.Evals))
	}

	eval := c.Evals[0]
	if eval.EvalID != "eval-1" {
		t.Errorf("expected eval ID %q, got %q", "eval-1", eval.EvalID)
	}
	if len(eval.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(eval.Variants))
	}

	// Control: pass rate unchanged (100% -> 100%), cost decreased, duration decreased.
	ctrl := eval.Variants[0]
	if ctrl.Status != StatusMatched {
		t.Errorf("expected matched, got %s", ctrl.Status)
	}
	if ctrl.PassRateDelta == nil {
		t.Fatal("expected pass rate delta")
	}
	if *ctrl.PassRateDelta != 0 {
		t.Errorf("expected 0pp delta, got %f", *ctrl.PassRateDelta)
	}
	if ctrl.CostDelta == nil || *ctrl.CostDelta >= 0 {
		t.Errorf("expected negative cost delta, got %v", ctrl.CostDelta)
	}
	if ctrl.DurationDeltaMs == nil || *ctrl.DurationDeltaMs >= 0 {
		t.Errorf("expected negative duration delta, got %v", ctrl.DurationDeltaMs)
	}

	// Variant: pass rate improved (50% -> 100%).
	variant := eval.Variants[1]
	if variant.PassRateDelta == nil {
		t.Fatal("expected pass rate delta for variant")
	}
	if *variant.PassRateDelta != 0.5 {
		t.Errorf("expected +0.5 pass rate delta, got %f", *variant.PassRateDelta)
	}
}

func TestCompare_MismatchedVariants(t *testing.T) {
	baseline := makeSuiteResult("baseline", []result.EvalResult{
		{
			EvalID: "eval-1",
			Variants: []result.VariantResult{
				{Name: "control", Runs: makeRuns([]bool{true}, 0.05, 5000)},
				{Name: "old-variant", Runs: makeRuns([]bool{false}, 0.10, 8000)},
			},
		},
	})

	candidate := makeSuiteResult("candidate", []result.EvalResult{
		{
			EvalID: "eval-1",
			Variants: []result.VariantResult{
				{Name: "control", Runs: makeRuns([]bool{true}, 0.05, 5000)},
				{Name: "new-variant", Runs: makeRuns([]bool{true}, 0.03, 3000)},
			},
		},
	})

	c := Compare(baseline, candidate)
	eval := c.Evals[0]

	if len(eval.Variants) != 3 {
		t.Fatalf("expected 3 variants (matched + removed + added), got %d", len(eval.Variants))
	}

	statusMap := map[string]ComparisonStatus{}
	for _, tc := range eval.Variants {
		statusMap[tc.Name] = tc.Status
	}

	if statusMap["control"] != StatusMatched {
		t.Errorf("expected control to be matched, got %s", statusMap["control"])
	}
	if statusMap["old-variant"] != StatusRemoved {
		t.Errorf("expected old-variant to be removed, got %s", statusMap["old-variant"])
	}
	if statusMap["new-variant"] != StatusAdded {
		t.Errorf("expected new-variant to be added, got %s", statusMap["new-variant"])
	}
}

func TestCompare_MismatchedEvals(t *testing.T) {
	baseline := makeSuiteResult("baseline", []result.EvalResult{
		{EvalID: "eval-1", Variants: []result.VariantResult{{Name: "control", Runs: makeRuns([]bool{true}, 0.05, 5000)}}},
		{EvalID: "eval-old", Variants: []result.VariantResult{{Name: "control", Runs: makeRuns([]bool{true}, 0.05, 5000)}}},
	})

	candidate := makeSuiteResult("candidate", []result.EvalResult{
		{EvalID: "eval-1", Variants: []result.VariantResult{{Name: "control", Runs: makeRuns([]bool{true}, 0.05, 5000)}}},
		{EvalID: "eval-new", Variants: []result.VariantResult{{Name: "control", Runs: makeRuns([]bool{true}, 0.03, 3000)}}},
	})

	c := Compare(baseline, candidate)

	if len(c.Evals) != 3 {
		t.Fatalf("expected 3 evals, got %d", len(c.Evals))
	}

	evalMap := map[string]EvalComparison{}
	for _, e := range c.Evals {
		evalMap[e.EvalID] = e
	}

	// eval-1 should be matched with matched variants.
	if evalMap["eval-1"].Variants[0].Status != StatusMatched {
		t.Error("expected eval-1 variant to be matched")
	}

	// eval-old should have removed variants.
	if evalMap["eval-old"].Variants[0].Status != StatusRemoved {
		t.Error("expected eval-old variant to be removed")
	}

	// eval-new should have added variants.
	if evalMap["eval-new"].Variants[0].Status != StatusAdded {
		t.Error("expected eval-new variant to be added")
	}
}

func TestCompare_IdenticalResults(t *testing.T) {
	sr := makeSuiteResult("run", []result.EvalResult{
		{
			EvalID: "eval-1",
			Variants: []result.VariantResult{
				{Name: "control", Runs: makeRuns([]bool{true, false, true}, 0.05, 5000)},
			},
		},
	})

	c := Compare(sr, sr)

	tc := c.Evals[0].Variants[0]
	if tc.Status != StatusMatched {
		t.Errorf("expected matched, got %s", tc.Status)
	}
	if tc.PassRateDelta == nil || *tc.PassRateDelta != 0 {
		t.Errorf("expected 0 pass rate delta for identical runs, got %v", tc.PassRateDelta)
	}
	if tc.CostDelta == nil || *tc.CostDelta != 0 {
		t.Errorf("expected 0 cost delta for identical runs, got %v", tc.CostDelta)
	}
	if tc.DurationDeltaMs == nil || *tc.DurationDeltaMs != 0 {
		t.Errorf("expected 0 duration delta for identical runs, got %v", tc.DurationDeltaMs)
	}
}

func TestCompare_NoPassData(t *testing.T) {
	runs := []result.RunResult{
		{Sample: 1, CostUSD: 0.05, DurationMs: 5000},
	}
	baseline := makeSuiteResult("baseline", []result.EvalResult{
		{EvalID: "eval-1", Variants: []result.VariantResult{{Name: "t1", Runs: runs}}},
	})
	candidate := makeSuiteResult("candidate", []result.EvalResult{
		{EvalID: "eval-1", Variants: []result.VariantResult{{Name: "t1", Runs: runs}}},
	})

	c := Compare(baseline, candidate)
	tc := c.Evals[0].Variants[0]

	if tc.PassRateDelta != nil {
		t.Errorf("expected nil pass rate delta when no pass data, got %v", tc.PassRateDelta)
	}
	if tc.CostDelta == nil {
		t.Error("expected cost delta even without pass data")
	}
}

func TestCompare_RunMeta(t *testing.T) {
	baseline := makeSuiteResult("baseline run", nil)
	candidate := makeSuiteResult("candidate run", nil)

	c := Compare(baseline, candidate)

	if c.Baseline.Description != "baseline run" {
		t.Errorf("expected baseline description %q, got %q", "baseline run", c.Baseline.Description)
	}
	if c.Candidate.Description != "candidate run" {
		t.Errorf("expected candidate description %q, got %q", "candidate run", c.Candidate.Description)
	}
	if c.Baseline.StartedAt == "" {
		t.Error("expected non-empty baseline started_at")
	}
}

func TestCompare_PercentageDeltas(t *testing.T) {
	baseline := makeSuiteResult("baseline", []result.EvalResult{
		{
			EvalID: "eval-1",
			Variants: []result.VariantResult{
				{Name: "t1", Runs: makeRuns([]bool{true}, 0.10, 10000)},
			},
		},
	})
	candidate := makeSuiteResult("candidate", []result.EvalResult{
		{
			EvalID: "eval-1",
			Variants: []result.VariantResult{
				{Name: "t1", Runs: makeRuns([]bool{true}, 0.15, 12000)},
			},
		},
	})

	c := Compare(baseline, candidate)
	tc := c.Evals[0].Variants[0]

	if tc.CostDeltaPct == nil {
		t.Fatal("expected cost delta percentage")
	}
	if !approxEqual(*tc.CostDeltaPct, 50.0, 0.01) {
		t.Errorf("expected ~50%% cost increase, got %f", *tc.CostDeltaPct)
	}

	if tc.DurationDeltaPct == nil {
		t.Fatal("expected duration delta percentage")
	}
	if !approxEqual(*tc.DurationDeltaPct, 20.0, 0.01) {
		t.Errorf("expected ~20%% duration increase, got %f", *tc.DurationDeltaPct)
	}
}

func TestWriteMarkdown(t *testing.T) {
	c := &Comparison{
		Baseline:  RunMeta{Description: "baseline", StartedAt: "2026-04-18T10:00:00Z"},
		Candidate: RunMeta{Description: "candidate", StartedAt: "2026-04-18T11:00:00Z"},
		Evals: []EvalComparison{
			{
				EvalID:   "eval-1",
				EvalName: "Fizzbuzz",
				Variants: []VariantComparison{
					{Name: "control", Status: StatusMatched, PassRateDelta: float64Ptr(0.5)},
					{Name: "old", Status: StatusRemoved},
					{Name: "new", Status: StatusAdded},
				},
			},
		},
	}

	var buf bytes.Buffer
	WriteMarkdown(&buf, c)
	out := buf.String()

	for _, want := range []string{
		"# Comparison Report",
		"Baseline:",
		"Candidate:",
		"Fizzbuzz",
		"control",
		"matched",
		"old",
		"removed",
		"new",
		"added",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown output missing %q, got:\n%s", want, out)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	c := &Comparison{
		Baseline:  RunMeta{Description: "baseline", StartedAt: "2026-04-18T10:00:00Z"},
		Candidate: RunMeta{Description: "candidate", StartedAt: "2026-04-18T11:00:00Z"},
		Evals: []EvalComparison{
			{
				EvalID:   "eval-1",
				EvalName: "Test",
				Variants: []VariantComparison{
					{Name: "t1", Status: StatusMatched, CostDelta: float64Ptr(-0.01)},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, c); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed Comparison
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed.Baseline.Description != "baseline" {
		t.Errorf("expected baseline description, got %q", parsed.Baseline.Description)
	}
	if len(parsed.Evals) != 1 || len(parsed.Evals[0].Variants) != 1 {
		t.Fatal("unexpected JSON structure")
	}
	if parsed.Evals[0].Variants[0].CostDelta == nil || *parsed.Evals[0].Variants[0].CostDelta != -0.01 {
		t.Error("cost delta not preserved in JSON roundtrip")
	}
}

func TestWrite_InvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, &Comparison{}, "csv")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}
