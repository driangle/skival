package report

import (
	"math"
	"testing"

	"github.com/driangle/skival/internal/result"
)

func boolPtr(b bool) *bool { return &b }

func TestRankTreatments_Empty(t *testing.T) {
	sr := &result.SuiteResult{}
	ranks := RankTreatments(sr)
	if ranks != nil {
		t.Fatalf("expected nil, got %v", ranks)
	}
}

func TestRankTreatments_SingleTreatment(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{{
				Name: "control",
				Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 1000, Pass: boolPtr(true)},
				},
			}},
		}},
	}
	ranks := RankTreatments(sr)
	if len(ranks) != 1 {
		t.Fatalf("expected 1 rank, got %d", len(ranks))
	}
	if ranks[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", ranks[0].Rank)
	}
	// Single treatment: all normalized to 1.0, composite = 1.0
	if ranks[0].CompositeScore != 1.0 {
		t.Errorf("expected composite 1.0, got %f", ranks[0].CompositeScore)
	}
}

func TestRankTreatments_BestTreatmentRanksFirst(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{
					Name: "expensive-slow-failing",
					Runs: []result.RunResult{
						{CostUSD: 10.0, DurationMs: 5000, Pass: boolPtr(false)},
						{CostUSD: 10.0, DurationMs: 5000, Pass: boolPtr(false)},
					},
				},
				{
					Name: "cheap-fast-passing",
					Runs: []result.RunResult{
						{CostUSD: 1.0, DurationMs: 500, Pass: boolPtr(true)},
						{CostUSD: 1.0, DurationMs: 500, Pass: boolPtr(true)},
					},
				},
			},
		}},
	}
	ranks := RankTreatments(sr)
	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(ranks))
	}
	if ranks[0].Name != "cheap-fast-passing" {
		t.Errorf("expected 'cheap-fast-passing' first, got %q", ranks[0].Name)
	}
	if ranks[0].Rank != 1 || ranks[1].Rank != 2 {
		t.Error("rank numbers incorrect")
	}
}

func TestRankTreatments_Ties(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{
					Name: "b",
					Runs: []result.RunResult{
						{CostUSD: 5.0, DurationMs: 1000, Pass: boolPtr(true)},
					},
				},
				{
					Name: "a",
					Runs: []result.RunResult{
						{CostUSD: 5.0, DurationMs: 1000, Pass: boolPtr(true)},
					},
				},
			},
		}},
	}
	ranks := RankTreatments(sr)
	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(ranks))
	}
	// Tied scores — deterministic by name
	if ranks[0].Name != "a" || ranks[1].Name != "b" {
		t.Errorf("expected alphabetical tiebreak: got %q, %q", ranks[0].Name, ranks[1].Name)
	}
	if ranks[0].CompositeScore != ranks[1].CompositeScore {
		t.Error("tied treatments should have equal scores")
	}
}

func TestRankTreatments_PassRateDominates(t *testing.T) {
	// Treatment A: 100% pass, expensive. Treatment B: 0% pass, cheap.
	// Pass weight (0.6) should dominate cost weight (0.28).
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{
					Name: "passes",
					Runs: []result.RunResult{
						{CostUSD: 100.0, DurationMs: 10000, Pass: boolPtr(true)},
					},
				},
				{
					Name: "fails",
					Runs: []result.RunResult{
						{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(false)},
					},
				},
			},
		}},
	}
	ranks := RankTreatments(sr)
	if ranks[0].Name != "passes" {
		t.Errorf("pass rate should dominate, got %q first", ranks[0].Name)
	}
}

func TestRankTreatments_MultipleEvals(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{
				Treatments: []result.TreatmentResult{
					{Name: "a", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(true)}}},
					{Name: "b", Runs: []result.RunResult{{CostUSD: 4.0, DurationMs: 400, Pass: boolPtr(true)}}},
				},
			},
			{
				Treatments: []result.TreatmentResult{
					{Name: "a", Runs: []result.RunResult{{CostUSD: 3.0, DurationMs: 300, Pass: boolPtr(true)}}},
					{Name: "b", Runs: []result.RunResult{{CostUSD: 5.0, DurationMs: 500, Pass: boolPtr(false)}}},
				},
			},
		},
	}
	ranks := RankTreatments(sr)
	if ranks[0].Name != "a" {
		t.Errorf("expected 'a' first (better pass rate + lower cost), got %q", ranks[0].Name)
	}
	// Treatment a: pass rate 2/2=1.0, median cost=2.5, median dur=250
	// Treatment b: pass rate 1/2=0.5, median cost=4.5, median dur=450
	if ranks[0].PassRate != 1.0 {
		t.Errorf("expected pass rate 1.0, got %f", ranks[0].PassRate)
	}
	if ranks[1].PassRate != 0.5 {
		t.Errorf("expected pass rate 0.5, got %f", ranks[1].PassRate)
	}
}

func TestRankTreatments_UnverifiedRuns(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{{
				Name: "unverified",
				Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 100, Pass: nil},
					{CostUSD: 2.0, DurationMs: 200, Pass: nil},
				},
			}},
		}},
	}
	ranks := RankTreatments(sr)
	if ranks[0].PassRate != 0 {
		t.Errorf("expected 0 pass rate for unverified, got %f", ranks[0].PassRate)
	}
}

func TestWeightsSum(t *testing.T) {
	sum := WeightPass + WeightCost + WeightDuration
	if math.Abs(sum-1.0) > 1e-10 {
		t.Errorf("weights sum to %f, expected 1.0", sum)
	}
}

func TestNormHigherBetter(t *testing.T) {
	if v := normHigherBetter(10, 0, 10); v != 1.0 {
		t.Errorf("max should be 1.0, got %f", v)
	}
	if v := normHigherBetter(0, 0, 10); v != 0.0 {
		t.Errorf("min should be 0.0, got %f", v)
	}
	if v := normHigherBetter(5, 5, 5); v != 1.0 {
		t.Errorf("equal should be 1.0, got %f", v)
	}
}

func TestNormLowerBetter(t *testing.T) {
	if v := normLowerBetter(0, 0, 10); v != 1.0 {
		t.Errorf("min should be 1.0, got %f", v)
	}
	if v := normLowerBetter(10, 0, 10); v != 0.0 {
		t.Errorf("max should be 0.0, got %f", v)
	}
	if v := normLowerBetter(5, 5, 5); v != 1.0 {
		t.Errorf("equal should be 1.0, got %f", v)
	}
}
