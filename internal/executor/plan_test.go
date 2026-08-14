package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/color"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

func planTestSuite() *suite.Suite {
	return &suite.Suite{
		Description: "test suite",
		Evals: []suite.Eval{
			{
				ID:      "eval-1",
				Name:    "First Eval",
				Samples: intPtr(3),
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code", Model: "claude-opus-4-6"},
					{Name: "variation", Runner: "claude-code", Model: "claude-sonnet-4-6"},
				},
			},
			{
				ID:   "eval-2",
				Name: "Second Eval",
				Variants: []suite.Variant{
					{Name: "control", Runner: "exec"},
				},
			},
		},
	}
}

func TestBuildPlan_MatrixAndSamples(t *testing.T) {
	plan, err := BuildPlan(planTestSuite(), &Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.TotalCells != 3 {
		t.Errorf("expected 3 cells, got %d", plan.TotalCells)
	}
	// eval-1: 2 variants × 3 samples = 6; eval-2: 1 variant × 1 sample = 1.
	if plan.TotalSamples != 7 {
		t.Errorf("expected 7 total samples, got %d", plan.TotalSamples)
	}

	first := plan.Entries[0]
	if first.EvalName != "First Eval" || first.Variant != "control" {
		t.Errorf("unexpected first entry: %+v", first)
	}
	if first.Model != "claude-opus-4-6" || first.Samples != 3 {
		t.Errorf("first entry model/samples wrong: %+v", first)
	}
	// eval-2 has no eval-level samples, so it resolves to the default of 1.
	last := plan.Entries[2]
	if last.Samples != 1 || last.Runner != "exec" {
		t.Errorf("expected eval-2 to resolve to 1 sample on exec, got %+v", last)
	}
}

func TestBuildPlan_SamplesOverride(t *testing.T) {
	plan, err := BuildPlan(planTestSuite(), &Options{Samples: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range plan.Entries {
		if e.Samples != 2 {
			t.Errorf("--samples override not applied to %s/%s: got %d", e.EvalID, e.Variant, e.Samples)
		}
	}
	if plan.TotalSamples != 6 {
		t.Errorf("expected 3 cells × 2 samples = 6, got %d", plan.TotalSamples)
	}
}

func TestBuildPlan_FiltersAndInvalidFilter(t *testing.T) {
	plan, err := BuildPlan(planTestSuite(), &Options{EvalIDs: []string{"eval-2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.TotalCells != 1 || plan.Entries[0].EvalID != "eval-2" {
		t.Errorf("eval filter not applied: %+v", plan.Entries)
	}

	if _, err := BuildPlan(planTestSuite(), &Options{EvalIDs: []string{"nope"}}); err == nil {
		t.Error("expected error for unknown eval filter")
	}
}

func TestApplyEstimate_FromPriors(t *testing.T) {
	plan, _ := BuildPlan(planTestSuite(), &Options{})

	prior := &result.SuiteResult{
		Evals: []result.EvalResult{
			{
				EvalID: "eval-1",
				Variants: []result.VariantResult{
					{Name: "control", Aggregate: &result.Aggregate{MedianCostUSD: 0.10}},
					{Name: "variation", Aggregate: &result.Aggregate{MedianCostUSD: 0.20}},
				},
			},
			// eval-2 has no prior aggregate on purpose: its cell stays unpriced.
		},
	}

	plan.ApplyEstimate(PriorsFromResults(prior))

	// eval-1/control: 0.10 × 3 = 0.30; eval-1/variation: 0.20 × 3 = 0.60.
	if plan.Entries[0].EstCostUSD == nil || !approxEqual(*plan.Entries[0].EstCostUSD, 0.30) {
		t.Errorf("control estimate wrong: %+v", plan.Entries[0].EstCostUSD)
	}
	if plan.Entries[1].EstCostUSD == nil || !approxEqual(*plan.Entries[1].EstCostUSD, 0.60) {
		t.Errorf("variation estimate wrong: %+v", plan.Entries[1].EstCostUSD)
	}
	if plan.Entries[2].EstCostUSD != nil {
		t.Errorf("eval-2 cell should be unpriced, got %+v", plan.Entries[2].EstCostUSD)
	}
	if plan.EstimatedCells != 2 || plan.EstTotalUSD == nil {
		t.Fatalf("expected 2 priced cells with a total, got cells=%d total=%v", plan.EstimatedCells, plan.EstTotalUSD)
	}
	if !approxEqual(*plan.EstTotalUSD, 0.90) {
		t.Errorf("expected estimated total 0.90, got %f", *plan.EstTotalUSD)
	}
}

func approxEqual(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}

func TestApplyEstimate_NoPriorsLeavesTotalNil(t *testing.T) {
	plan, _ := BuildPlan(planTestSuite(), &Options{})
	plan.ApplyEstimate(PriorsFromResults(&result.SuiteResult{}))
	if plan.EstTotalUSD != nil {
		t.Errorf("expected nil estimated total with no priors, got %v", *plan.EstTotalUSD)
	}
	if plan.EstimatedCells != 0 {
		t.Errorf("expected 0 priced cells, got %d", plan.EstimatedCells)
	}
}

func TestWritePlan_Unpriced(t *testing.T) {
	color.SetEnabled(false)
	plan, _ := BuildPlan(planTestSuite(), &Options{})

	var buf bytes.Buffer
	WritePlan(&buf, plan)
	out := buf.String()

	if strings.Contains(out, "EST COST") {
		t.Error("unpriced plan should not show an EST COST column")
	}
	for _, want := range []string{"First Eval", "control", "claude-opus-4-6", "7 total samples", "dry run"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestWritePlan_Priced(t *testing.T) {
	color.SetEnabled(false)
	plan, _ := BuildPlan(planTestSuite(), &Options{})
	prior := &result.SuiteResult{
		Evals: []result.EvalResult{{
			EvalID: "eval-1",
			Variants: []result.VariantResult{
				{Name: "control", Aggregate: &result.Aggregate{MedianCostUSD: 0.10}},
				{Name: "variation", Aggregate: &result.Aggregate{MedianCostUSD: 0.20}},
			},
		}},
	}
	plan.ApplyEstimate(PriorsFromResults(prior))

	var buf bytes.Buffer
	WritePlan(&buf, plan)
	out := buf.String()

	if !strings.Contains(out, "EST COST") {
		t.Error("priced plan should show an EST COST column")
	}
	if !strings.Contains(out, "Estimated total cost: $0.9000") {
		t.Errorf("expected estimated total in output, got:\n%s", out)
	}
	if !strings.Contains(out, "covers 2 of 3 cells") {
		t.Errorf("expected partial-coverage note, got:\n%s", out)
	}
}
