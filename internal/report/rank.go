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
}

// DefaultWeights returns the default ranking weights.
func DefaultWeights() Weights {
	return Weights{
		Correctness: DefaultWeightPass,
		Cost:        DefaultWeightCost,
		Duration:    DefaultWeightDuration,
	}
}

// VariantRank holds the ranking data for a single variant.
type VariantRank struct {
	Name           string
	Runner         string
	Model          string
	PassRate       float64
	// MedianCostUSD and MedianDuration are the mean of the variant's per-eval
	// medians, not a single median pooled across every eval. Pooling would mix
	// the distributions of a cheap eval and an expensive eval into one number.
	MedianCostUSD  float64
	MedianDuration int64
	CompositeScore float64
	Rank           int
}

// RankVariants computes a weighted composite score for each variant across all
// evals in a suite result and returns them sorted best-first.
//
// Cost and duration are normalized *within each eval* (relative to that eval's
// best variant) and then averaged across evals, so the composite reflects the
// magnitude of each gap — a variant twice as expensive as the eval's best
// scores 0.5 on cost for that eval, not 0. Pass rate is already bounded to
// [0,1] and needs no normalization.
//
// Single-variant evals: the lone variant is the best on cost and duration by
// definition, so it scores 1.0 on both for that eval; only its pass rate can
// pull the composite below the weight sum.
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

// variantAccumulator gathers a variant's cross-eval data. Cost and duration are
// normalized within each eval before being summed, so per-eval distributions
// are never pooled into a single figure.
type variantAccumulator struct {
	runner   string
	model    string
	passed   int
	verified int

	evalCount   int
	costNormSum float64
	durNormSum  float64
	costMedSum  float64
	durMedSum   float64
}

func (a *variantAccumulator) toRank(name string, w Weights) VariantRank {
	passRate := 0.0
	if a.verified > 0 {
		passRate = float64(a.passed) / float64(a.verified)
	}

	var costNorm, durNorm, medCost, medDur float64
	if a.evalCount > 0 {
		costNorm = a.costNormSum / float64(a.evalCount)
		durNorm = a.durNormSum / float64(a.evalCount)
		medCost = a.costMedSum / float64(a.evalCount)
		medDur = a.durMedSum / float64(a.evalCount)
	}

	return VariantRank{
		Name:           name,
		Runner:         a.runner,
		Model:          a.model,
		PassRate:       passRate,
		MedianCostUSD:  medCost,
		MedianDuration: int64(medDur),
		CompositeScore: w.Correctness*passRate + w.Cost*costNorm + w.Duration*durNorm,
	}
}

// accumulate walks every eval, scoring cost/duration relative to the eval's own
// best variant, and folds the results into one accumulator per variant.
func accumulate(sr *result.SuiteResult) map[string]*variantAccumulator {
	accs := make(map[string]*variantAccumulator)
	for _, eval := range sr.Evals {
		scoreEval(eval, accs)
	}
	return accs
}

// evalMetric is a variant's within-eval median cost and duration.
type evalMetric struct {
	name string
	cost float64
	dur  float64
}

// scoreEval accumulates pass counts for every variant in the eval and adds each
// variant's cost/duration scored relative to the best variant in this eval.
func scoreEval(eval result.EvalResult, accs map[string]*variantAccumulator) {
	var metrics []evalMetric

	for _, v := range eval.Variants {
		a := accs[v.Name]
		if a == nil {
			a = &variantAccumulator{runner: v.Runner, model: v.Model}
			accs[v.Name] = a
		}

		for _, run := range v.Runs {
			if run.Pass != nil {
				a.verified++
				if *run.Pass {
					a.passed++
				}
			}
		}

		agg := result.ComputeAggregate(v.Runs)
		if agg == nil {
			continue
		}
		metrics = append(metrics, evalMetric{
			name: v.Name,
			cost: agg.MedianCostUSD,
			dur:  float64(agg.MedianDurationMs),
		})
	}

	if len(metrics) == 0 {
		return
	}

	bestCost, bestDur := metrics[0].cost, metrics[0].dur
	for _, m := range metrics[1:] {
		if m.cost < bestCost {
			bestCost = m.cost
		}
		if m.dur < bestDur {
			bestDur = m.dur
		}
	}

	for _, m := range metrics {
		a := accs[m.name]
		a.costNormSum += ratioLowerBetter(m.cost, bestCost)
		a.durNormSum += ratioLowerBetter(m.dur, bestDur)
		a.costMedSum += m.cost
		a.durMedSum += m.dur
		a.evalCount++
	}
}

// ratioLowerBetter scores a value against the best (lowest) value among the
// variants in an eval, for metrics where lower is better (cost, duration). The
// best value scores 1.0; a value twice the best scores 0.5, giving the
// composite sensitivity to the magnitude of the gap rather than only the
// ordering. A zero or negative value is treated as the best possible; if the
// best is zero while this value is positive, it scores 0.0.
func ratioLowerBetter(val, best float64) float64 {
	if val <= 0 {
		return 1.0
	}
	if best <= 0 {
		return 0.0
	}
	return best / val
}
