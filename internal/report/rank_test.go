package report

import (
	"math"
	"testing"

	"github.com/driangle/skival/internal/result"
)

func boolPtr(b bool) *bool { return &b }

func TestRankVariants_Empty(t *testing.T) {
	sr := &result.SuiteResult{}
	ranks := RankVariants(sr, DefaultWeights())
	if ranks != nil {
		t.Fatalf("expected nil, got %v", ranks)
	}
}

func TestRankVariants_SingleVariant(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{{
				Name: "control",
				Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 1000, Pass: boolPtr(true)},
				},
			}},
		}},
	}
	ranks := RankVariants(sr, DefaultWeights())
	if len(ranks) != 1 {
		t.Fatalf("expected 1 rank, got %d", len(ranks))
	}
	if ranks[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", ranks[0].Rank)
	}
	// Single variant: all normalized to 1.0, composite = 1.0
	if ranks[0].CompositeScore != 1.0 {
		t.Errorf("expected composite 1.0, got %f", ranks[0].CompositeScore)
	}
}

func TestRankVariants_BestVariantRanksFirst(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
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
	ranks := RankVariants(sr, DefaultWeights())
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

func TestRankVariants_Ties(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
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
	ranks := RankVariants(sr, DefaultWeights())
	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(ranks))
	}
	// Tied scores — deterministic by name
	if ranks[0].Name != "a" || ranks[1].Name != "b" {
		t.Errorf("expected alphabetical tiebreak: got %q, %q", ranks[0].Name, ranks[1].Name)
	}
	if ranks[0].CompositeScore != ranks[1].CompositeScore {
		t.Error("tied variants should have equal scores")
	}
}

func TestRankVariants_PassRateDominates(t *testing.T) {
	// Variant A: 100% pass, expensive. Variant B: 0% pass, cheap.
	// Pass weight (0.6) should dominate cost weight (0.28).
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
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
	ranks := RankVariants(sr, DefaultWeights())
	if ranks[0].Name != "passes" {
		t.Errorf("pass rate should dominate, got %q first", ranks[0].Name)
	}
}

func TestRankVariants_MultipleEvals(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{
				Variants: []result.VariantResult{
					{Name: "a", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(true)}}},
					{Name: "b", Runs: []result.RunResult{{CostUSD: 4.0, DurationMs: 400, Pass: boolPtr(true)}}},
				},
			},
			{
				Variants: []result.VariantResult{
					{Name: "a", Runs: []result.RunResult{{CostUSD: 3.0, DurationMs: 300, Pass: boolPtr(true)}}},
					{Name: "b", Runs: []result.RunResult{{CostUSD: 5.0, DurationMs: 500, Pass: boolPtr(false)}}},
				},
			},
		},
	}
	ranks := RankVariants(sr, DefaultWeights())
	if ranks[0].Name != "a" {
		t.Errorf("expected 'a' first (better pass rate + lower cost), got %q", ranks[0].Name)
	}
	// Variant a: pass rate 2/2=1.0, median cost=2.5, median dur=250
	// Variant b: pass rate 1/2=0.5, median cost=4.5, median dur=450
	if ranks[0].PassRate != 1.0 {
		t.Errorf("expected pass rate 1.0, got %f", ranks[0].PassRate)
	}
	if ranks[1].PassRate != 0.5 {
		t.Errorf("expected pass rate 0.5, got %f", ranks[1].PassRate)
	}
}

func TestRankVariants_UnverifiedRuns(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{{
				Name: "unverified",
				Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 100, Pass: nil},
					{CostUSD: 2.0, DurationMs: 200, Pass: nil},
				},
			}},
		}},
	}
	ranks := RankVariants(sr, DefaultWeights())
	if ranks[0].PassRate != 0 {
		t.Errorf("expected 0 pass rate for unverified, got %f", ranks[0].PassRate)
	}
}

func TestWeightsSum(t *testing.T) {
	sum := DefaultWeightPass + DefaultWeightCost + DefaultWeightDuration
	if math.Abs(sum-1.0) > 1e-10 {
		t.Errorf("weights sum to %f, expected 1.0", sum)
	}
}

func TestRankVariants_CustomWeightsCostDominates(t *testing.T) {
	// With cost weight at 0.90, a cheap-but-failing variant should rank above
	// an expensive-but-passing variant.
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{
					Name: "expensive-passing",
					Runs: []result.RunResult{
						{CostUSD: 100.0, DurationMs: 1000, Pass: boolPtr(true)},
					},
				},
				{
					Name: "cheap-failing",
					Runs: []result.RunResult{
						{CostUSD: 0.01, DurationMs: 1000, Pass: boolPtr(false)},
					},
				},
			},
		}},
	}

	costHeavy := Weights{Correctness: 0.05, Cost: 0.90, Duration: 0.05}
	ranks := RankVariants(sr, costHeavy)
	if ranks[0].Name != "cheap-failing" {
		t.Errorf("with cost weight 0.90, cheap variant should rank first, got %q", ranks[0].Name)
	}

	// With default weights, the passing variant should rank first.
	defaultRanks := RankVariants(sr, DefaultWeights())
	if defaultRanks[0].Name != "expensive-passing" {
		t.Errorf("with default weights, passing variant should rank first, got %q", defaultRanks[0].Name)
	}
}

func TestRankVariants_CustomWeightsDurationDominates(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{
					Name: "fast-failing",
					Runs: []result.RunResult{
						{CostUSD: 5.0, DurationMs: 100, Pass: boolPtr(false)},
					},
				},
				{
					Name: "slow-passing",
					Runs: []result.RunResult{
						{CostUSD: 5.0, DurationMs: 10000, Pass: boolPtr(true)},
					},
				},
			},
		}},
	}

	durationHeavy := Weights{Correctness: 0.05, Cost: 0.05, Duration: 0.90}
	ranks := RankVariants(sr, durationHeavy)
	if ranks[0].Name != "fast-failing" {
		t.Errorf("with duration weight 0.90, fast variant should rank first, got %q", ranks[0].Name)
	}
}

func TestRankVariants_ZeroWeight(t *testing.T) {
	// With correctness weight at 0, pass rate shouldn't affect ranking.
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{
					Name: "all-pass-expensive",
					Runs: []result.RunResult{
						{CostUSD: 100.0, DurationMs: 5000, Pass: boolPtr(true)},
					},
				},
				{
					Name: "all-fail-cheap",
					Runs: []result.RunResult{
						{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(false)},
					},
				},
			},
		}},
	}

	noCorrWeight := Weights{Correctness: 0, Cost: 0.50, Duration: 0.50}
	ranks := RankVariants(sr, noCorrWeight)
	if ranks[0].Name != "all-fail-cheap" {
		t.Errorf("with correctness=0, cheap/fast variant should rank first, got %q", ranks[0].Name)
	}
}

func TestRatioLowerBetter(t *testing.T) {
	if v := ratioLowerBetter(5, 5); v != 1.0 {
		t.Errorf("best value should be 1.0, got %f", v)
	}
	if v := ratioLowerBetter(10, 5); v != 0.5 {
		t.Errorf("twice the best should be 0.5, got %f", v)
	}
	if v := ratioLowerBetter(0, 0); v != 1.0 {
		t.Errorf("zero value should be 1.0, got %f", v)
	}
	if v := ratioLowerBetter(5, 0); v != 0.0 {
		t.Errorf("positive value vs zero best should be 0.0, got %f", v)
	}
}

// TestRankVariants_MagnitudeSensitivity is the flagship control + 1 treatment
// case: with only two variants, min-max normalization would score the loser 0.0
// regardless of whether it lost by 1% or 90%. Ratio-to-best must instead
// separate a small gap from a large one.
func TestRankVariants_MagnitudeSensitivity(t *testing.T) {
	mk := func(treatmentCost float64) *result.SuiteResult {
		return &result.SuiteResult{
			Evals: []result.EvalResult{{
				Variants: []result.VariantResult{
					{Name: "control", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 1000, Pass: boolPtr(true)}}},
					{Name: "treatment", Runs: []result.RunResult{{CostUSD: treatmentCost, DurationMs: 1000, Pass: boolPtr(true)}}},
				},
			}},
		}
	}

	find := func(ranks []VariantRank, name string) VariantRank {
		for _, r := range ranks {
			if r.Name == name {
				return r
			}
		}
		t.Fatalf("variant %q not found", name)
		return VariantRank{}
	}

	smallGap := RankVariants(mk(1.01), DefaultWeights()) // treatment 1% pricier
	largeGap := RankVariants(mk(10.0), DefaultWeights()) // treatment 10x pricier

	smallTreatment := find(smallGap, "treatment").CompositeScore
	largeTreatment := find(largeGap, "treatment").CompositeScore

	if smallTreatment <= largeTreatment {
		t.Errorf("magnitude should matter: 1%% gap score (%f) must exceed 90%% gap score (%f)",
			smallTreatment, largeTreatment)
	}
	// The cheaper control still outranks the treatment in both suites.
	if find(smallGap, "control").CompositeScore <= smallTreatment {
		t.Error("control should outrank the pricier treatment")
	}
}

// TestRankVariants_PerEvalNotPooled proves cost/duration are aggregated as the
// mean of per-eval medians rather than a single median pooled across every run.
// The cheap eval has three runs and the expensive eval one, so a global pooled
// median would be dominated by the cheap runs (~1); per-eval aggregation yields
// the mean of the two eval medians instead.
func TestRankVariants_PerEvalNotPooled(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{Variants: []result.VariantResult{{Name: "a", Runs: []result.RunResult{
				{CostUSD: 1, DurationMs: 10, Pass: boolPtr(true)},
				{CostUSD: 1, DurationMs: 10, Pass: boolPtr(true)},
				{CostUSD: 1, DurationMs: 10, Pass: boolPtr(true)},
			}}}},
			{Variants: []result.VariantResult{{Name: "a", Runs: []result.RunResult{
				{CostUSD: 100, DurationMs: 1000, Pass: boolPtr(true)},
			}}}},
		},
	}
	ranks := RankVariants(sr, DefaultWeights())
	if len(ranks) != 1 {
		t.Fatalf("expected 1 rank, got %d", len(ranks))
	}
	// mean of per-eval medians: (1 + 100) / 2 = 50.5, not the pooled median (~1).
	if math.Abs(ranks[0].MedianCostUSD-50.5) > 1e-9 {
		t.Errorf("expected mean-of-per-eval-median cost 50.5, got %f", ranks[0].MedianCostUSD)
	}
	// duration: (10 + 1000) / 2 = 505.
	if ranks[0].MedianDuration != 505 {
		t.Errorf("expected mean-of-per-eval-median duration 505, got %d", ranks[0].MedianDuration)
	}
}

// TestRankVariants_PerEvalNormalizationSymmetry: A wins eval1 on cost by 2x and
// B wins eval2 on cost by 2x. Because each eval is normalized on its own scale,
// their cost contributions are equal and the composites tie — global pooling of
// raw costs across the two different scales would instead pick a winner.
func TestRankVariants_PerEvalNormalizationSymmetry(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 1000, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 2, DurationMs: 1000, Pass: boolPtr(true)}}},
			}},
			{Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 200, DurationMs: 1000, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 100, DurationMs: 1000, Pass: boolPtr(true)}}},
			}},
		},
	}
	ranks := RankVariants(sr, DefaultWeights())
	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(ranks))
	}
	if math.Abs(ranks[0].CompositeScore-ranks[1].CompositeScore) > 1e-9 {
		t.Errorf("symmetric per-eval wins should tie, got %f and %f",
			ranks[0].CompositeScore, ranks[1].CompositeScore)
	}
}
