package report

import (
	"math"
	"testing"

	"github.com/driangle/skival/internal/result"
)

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

// TestRankVariants_QualityWeightBreaksTie: two variants pass identically on
// cost/duration; the comparative quality score, with a non-zero quality weight,
// promotes the higher-quality variant.
func TestRankVariants_QualityWeightBreaksTie(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 5.0, DurationMs: 1000, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 5.0, DurationMs: 1000, Pass: boolPtr(true)}}},
			},
			Comparison: &result.Comparison{
				Scores: []result.ComparativeScore{
					{Variant: "a", Rating: 2, Score: 0.4},
					{Variant: "b", Rating: 5, Score: 1.0},
				},
			},
		}},
	}

	w := Weights{Correctness: 0.5, Cost: 0.14, Duration: 0.06, Quality: 0.30}
	ranks := RankVariants(sr, w)
	if ranks[0].Name != "b" {
		t.Errorf("higher-quality variant should rank first, got %q", ranks[0].Name)
	}
	if math.Abs(find2(ranks, "b").QualityScore-1.0) > 1e-9 {
		t.Errorf("b quality score = %f, want 1.0", find2(ranks, "b").QualityScore)
	}
	if math.Abs(find2(ranks, "a").QualityScore-0.4) > 1e-9 {
		t.Errorf("a quality score = %f, want 0.4", find2(ranks, "a").QualityScore)
	}
}

// TestRankVariants_QualityIgnoredWhenWeightZero: comparative scores must not
// move ranking when the quality weight is 0 (the default), preserving today's
// behavior for suites that don't opt in.
func TestRankVariants_QualityIgnoredWhenWeightZero(t *testing.T) {
	mk := func() *result.SuiteResult {
		return &result.SuiteResult{
			Evals: []result.EvalResult{{
				Variants: []result.VariantResult{
					{Name: "cheap", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
					{Name: "pricey", Runs: []result.RunResult{{CostUSD: 9.0, DurationMs: 900, Pass: boolPtr(true)}}},
				},
			}},
		}
	}
	withComparison := mk()
	withComparison.Evals[0].Comparison = &result.Comparison{
		Scores: []result.ComparativeScore{
			{Variant: "cheap", Rating: 1, Score: 0.2},
			{Variant: "pricey", Rating: 5, Score: 1.0},
		},
	}

	base := RankVariants(mk(), DefaultWeights())
	withCmp := RankVariants(withComparison, DefaultWeights())
	if base[0].Name != withCmp[0].Name {
		t.Errorf("comparison changed ranking despite quality weight 0: %q vs %q", base[0].Name, withCmp[0].Name)
	}
	if math.Abs(base[0].CompositeScore-withCmp[0].CompositeScore) > 1e-9 {
		t.Error("composite score changed despite quality weight 0")
	}
}

// TestRankVariants_QualityAveragesOnlyScoredEvals: a variant compared in one
// eval but not another averages quality only over the eval where it was scored.
func TestRankVariants_QualityAveragesOnlyScoredEvals(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{
				Variants: []result.VariantResult{
					{Name: "a", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
					{Name: "b", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
				},
				Comparison: &result.Comparison{Scores: []result.ComparativeScore{
					{Variant: "a", Rating: 4, Score: 0.8},
					{Variant: "b", Rating: 3, Score: 0.6},
				}},
			},
			{
				// Second eval has no comparison (e.g. one variant failed).
				Variants: []result.VariantResult{
					{Name: "a", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
				},
			},
		},
	}
	ranks := RankVariants(sr, Weights{Correctness: 0.7, Cost: 0.1, Duration: 0.1, Quality: 0.1})
	// a's quality is 0.8 (only the scored eval counts), not 0.4 (0.8 averaged with a 0).
	if math.Abs(find2(ranks, "a").QualityScore-0.8) > 1e-9 {
		t.Errorf("a quality = %f, want 0.8 (averaged only over scored evals)", find2(ranks, "a").QualityScore)
	}
}

// TestRankVariants_SkippedComparisonContributesNothing: a comparison marked
// skipped adds no quality signal.
func TestRankVariants_SkippedComparisonContributesNothing(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
			},
			Comparison: &result.Comparison{Skipped: "judge errored"},
		}},
	}
	ranks := RankVariants(sr, Weights{Correctness: 0.7, Cost: 0.1, Duration: 0.1, Quality: 0.1})
	for _, r := range ranks {
		if r.QualityScore != 0 {
			t.Errorf("%s quality = %f, want 0 for skipped comparison", r.Name, r.QualityScore)
		}
	}
}

func find2(ranks []VariantRank, name string) VariantRank {
	for _, r := range ranks {
		if r.Name == name {
			return r
		}
	}
	return VariantRank{}
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
