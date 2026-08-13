package report

import (
	"sort"

	"github.com/driangle/skival/internal/result"
)

const (
	DefaultWeightPass     = 0.60
	DefaultWeightCost     = 0.28
	DefaultWeightDuration = 0.12
)

// Weights defines the relative importance of each metric in the composite score.
type Weights struct {
	Correctness float64
	Cost        float64
	Duration    float64
	// Quality weights the comparative-judge quality signal. It defaults to 0,
	// so suites that do not use comparative judging rank exactly as before.
	Quality float64
	// Tokens weights median total token usage (input + output). It defaults to
	// 0, so suites that do not opt in rank exactly as before. Cost and Tokens
	// weigh the same (correlated) economic signal, so a suite normally sets one
	// or the other, not both.
	Tokens float64
}

// DefaultWeights returns the default ranking weights. Quality is 0 so ranking
// behavior is unchanged unless comparative judging is enabled.
func DefaultWeights() Weights {
	return Weights{
		Correctness: DefaultWeightPass,
		Cost:        DefaultWeightCost,
		Duration:    DefaultWeightDuration,
	}
}

// VariantRank holds the ranking data for a single variant.
type VariantRank struct {
	Name     string
	Runner   string
	Model    string
	PassRate float64
	// MedianCostUSD and MedianDuration are the mean of the variant's per-eval
	// medians, not a single median pooled across every eval. Pooling would mix
	// the distributions of a cheap eval and an expensive eval into one number.
	MedianCostUSD  float64
	MedianDuration int64
	// MedianTotalTokens is the mean of the variant's per-eval median total
	// tokens (input + output), matching how MedianCostUSD/MedianDuration are
	// aggregated. 0 when no run reported usage.
	MedianTotalTokens int64
	// QualityScore is the variant's mean comparative-judge score across the
	// evals where it was compared, on a [0,1] scale. 0 when never compared.
	QualityScore   float64
	CompositeScore float64
	Rank           int
}

// hasComparison reports whether any eval produced comparative quality scores.
// Reports use it to decide whether to show quality columns and sections.
func hasComparison(sr *result.SuiteResult) bool {
	for _, eval := range sr.Evals {
		if eval.Comparison != nil && len(eval.Comparison.Scores) > 0 {
			return true
		}
	}
	return false
}

// RankVariants computes a weighted composite score for each variant across all
// evals in a suite result and returns them sorted best-first.
//
// Cost, duration, and total tokens are normalized *within each eval* (relative
// to that eval's best variant) and then averaged across evals, so the composite
// reflects the magnitude of each gap — a variant twice as expensive as the
// eval's best scores 0.5 on cost for that eval, not 0. Pass rate is already
// bounded to [0,1] and needs no normalization.
//
// The token term contributes only when w.Tokens > 0. A variant that reported no
// usage has 0 total tokens and is treated as the best (mirroring how cost = 0 is
// handled), so a suite mixing runners that report tokens with runners that don't
// degrades gracefully rather than crashing or scoring NaN.
//
// Single-variant evals: the lone variant is the best on cost, duration, and
// tokens by definition, so it scores 1.0 on each for that eval; only its pass
// rate can pull the composite below the weight sum.
func RankVariants(sr *result.SuiteResult, w Weights) []VariantRank {
	accs := accumulate(sr)
	if len(accs) == 0 {
		return nil
	}

	ranks := make([]VariantRank, 0, len(accs))
	for name, a := range accs {
		ranks = append(ranks, a.toRank(name, w))
	}

	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].CompositeScore != ranks[j].CompositeScore {
			return ranks[i].CompositeScore > ranks[j].CompositeScore
		}
		return ranks[i].Name < ranks[j].Name
	})

	for i := range ranks {
		ranks[i].Rank = i + 1
	}

	return ranks
}
