package report

import (
	"math"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
)

// tokenRun builds a run with the given input/output tokens and a passing verdict,
// holding cost and duration constant so only the token signal varies.
func tokenRun(in, out int) result.RunResult {
	return result.RunResult{
		CostUSD:    1.0,
		DurationMs: 1000,
		Pass:       boolPtr(true),
		Usage:      agentrunner.Usage{InputTokens: in, OutputTokens: out},
	}
}

func TestRankVariants_TokensDimensionOrders(t *testing.T) {
	// Cost, duration, and pass rate are identical; only token usage differs. With
	// a token weight, the terse variant must rank first.
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "token-hungry", Runs: []result.RunResult{tokenRun(9000, 1000)}},
				{Name: "terse", Runs: []result.RunResult{tokenRun(900, 100)}},
			},
		}},
	}

	w := Weights{Correctness: 0.5, Duration: 0.2, Tokens: 0.3}
	ranks := RankVariants(sr, w)
	if ranks[0].Name != "terse" {
		t.Errorf("with a token weight, terse variant should rank first, got %q", ranks[0].Name)
	}

	// terse: 1000 total tokens (best) -> tokenNorm 1.0
	// hungry: 10000 total tokens -> tokenNorm 0.1
	// pass and duration are equal, so the composite gap is entirely the token term.
	wantGap := 0.3 * (1.0 - 0.1)
	gotGap := ranks[0].CompositeScore - ranks[1].CompositeScore
	if math.Abs(gotGap-wantGap) > 1e-9 {
		t.Errorf("composite gap = %g, want %g", gotGap, wantGap)
	}
}

func TestRankVariants_DefaultIgnoresTokens(t *testing.T) {
	// Two variants identical on cost/duration/pass but wildly different on tokens.
	// With the default weights (Tokens == 0) they must tie exactly — token usage
	// contributes nothing, so existing suites rank byte-for-byte as before.
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{tokenRun(100, 10)}},
				{Name: "b", Runs: []result.RunResult{tokenRun(100000, 10000)}},
			},
		}},
	}

	ranks := RankVariants(sr, DefaultWeights())
	if ranks[0].CompositeScore != ranks[1].CompositeScore {
		t.Errorf("default weights should ignore tokens: composites differ (%g vs %g)",
			ranks[0].CompositeScore, ranks[1].CompositeScore)
	}
}

func TestRankVariants_MedianTotalTokensReported(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{{
				Name: "v",
				Runs: []result.RunResult{tokenRun(1500, 240)},
			}},
		}},
	}

	ranks := RankVariants(sr, DefaultWeights())
	if got := ranks[0].MedianTotalTokens; got != 1740 {
		t.Errorf("MedianTotalTokens = %d, want 1740 (input+output)", got)
	}
}

func TestRankVariants_ZeroTokensDegradeGracefully(t *testing.T) {
	// A suite ranking on tokens where one runner reports no usage: the token-less
	// variant is treated as the best (like cost = 0), and no score is NaN/Inf.
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "reports-tokens", Runs: []result.RunResult{tokenRun(5000, 500)}},
				{Name: "no-usage", Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 1000, Pass: boolPtr(true)},
				}},
			},
		}},
	}

	w := Weights{Correctness: 0.5, Duration: 0.2, Tokens: 0.3}
	ranks := RankVariants(sr, w)
	for _, r := range ranks {
		if math.IsNaN(r.CompositeScore) || math.IsInf(r.CompositeScore, 0) {
			t.Fatalf("variant %q produced non-finite composite %g", r.Name, r.CompositeScore)
		}
	}
	if ranks[0].Name != "no-usage" {
		t.Errorf("token-less variant should score as most efficient, got %q first", ranks[0].Name)
	}
	if ranks[0].MedianTotalTokens != 0 {
		t.Errorf("token-less variant should report 0 total tokens, got %d", ranks[0].MedianTotalTokens)
	}
}

func TestRankVariants_AllZeroTokensNoDifferentiation(t *testing.T) {
	// When no variant in an eval reports tokens, the token term is uniform (all
	// best), so ranking falls back to the other dimensions without crashing.
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "slow", Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 9000, Pass: boolPtr(true)},
				}},
				{Name: "fast", Runs: []result.RunResult{
					{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)},
				}},
			},
		}},
	}

	w := Weights{Correctness: 0.5, Duration: 0.2, Tokens: 0.3}
	ranks := RankVariants(sr, w)
	if ranks[0].Name != "fast" {
		t.Errorf("with all-zero tokens, duration should decide: got %q first", ranks[0].Name)
	}
	// Both score the token term at 1.0, so the composite gap is the duration term only.
	wantGap := 0.2 * (1.0 - 100.0/9000.0)
	gotGap := ranks[0].CompositeScore - ranks[1].CompositeScore
	if math.Abs(gotGap-wantGap) > 1e-9 {
		t.Errorf("composite gap = %g, want %g", gotGap, wantGap)
	}
}
