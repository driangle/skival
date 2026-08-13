package report

import (
	"testing"

	"github.com/driangle/skival/internal/result"
)

// TestRankVariants_WilsonBounds checks that RankVariants populates a plausible
// 95% Wilson interval bracketing each variant's observed pass rate.
func TestRankVariants_WilsonBounds(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{
					Name: "a",
					Runs: []result.RunResult{
						{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)},
						{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)},
						{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(false)},
					},
				},
				{
					Name: "b",
					Runs: []result.RunResult{
						{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)},
						{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)},
						{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)},
					},
				},
			},
		}},
	}

	ranks := RankVariants(sr, DefaultWeights())
	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(ranks))
	}
	const eps = 1e-9
	for _, r := range ranks {
		if !(r.PassLow <= r.PassRate+eps && r.PassRate <= r.PassHigh+eps) {
			t.Errorf("%s: expected PassLow<=PassRate<=PassHigh, got %f<=%f<=%f",
				r.Name, r.PassLow, r.PassRate, r.PassHigh)
		}
		if r.PassLow < 0 || r.PassHigh > 1 {
			t.Errorf("%s: interval out of [0,1]: (%f,%f)", r.Name, r.PassLow, r.PassHigh)
		}
	}
}
