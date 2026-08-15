package report

import (
	"sort"

	"github.com/driangle/skival/internal/result"
)

// variantAccumulator gathers a variant's cross-eval data. Cost, duration, and
// tokens are normalized within each eval before being summed, so per-eval
// distributions are never pooled into a single figure.
type variantAccumulator struct {
	runner   string
	model    string
	passed   int
	verified int

	evalCount    int
	costNormSum  float64
	durNormSum   float64
	tokenNormSum float64
	costMedSum   float64
	durMedSum    float64
	tokenMedSum  float64

	// Quality is accumulated only over evals where this variant was compared,
	// so a variant that failed (and was excluded from comparison) is not
	// double-penalized on quality — its low pass rate already reflects that.
	qualSum   float64
	qualCount int

	// toolCounts tallies every tool the variant invoked across the whole suite,
	// keyed by tool name. Lazily created on first tool use.
	toolCounts map[string]int
}

// addToolCounts folds a single run's per-tool counts into the accumulator.
func (a *variantAccumulator) addToolCounts(counts map[string]int) {
	for name, n := range counts {
		if a.toolCounts == nil {
			a.toolCounts = make(map[string]int)
		}
		a.toolCounts[name] += n
	}
}

// sortedTools returns the accumulated tool census sorted by count desc, then
// name asc for stable output. Empty when no tools were used.
func (a *variantAccumulator) sortedTools() []ToolCount {
	if len(a.toolCounts) == 0 {
		return nil
	}
	tools := make([]ToolCount, 0, len(a.toolCounts))
	for name, n := range a.toolCounts {
		tools = append(tools, ToolCount{Name: name, Count: n})
	}
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Count != tools[j].Count {
			return tools[i].Count > tools[j].Count
		}
		return tools[i].Name < tools[j].Name
	})
	return tools
}

func (a *variantAccumulator) toRank(name string, w Weights) VariantRank {
	passRate := 0.0
	if a.verified > 0 {
		passRate = float64(a.passed) / float64(a.verified)
	}
	passLow, passHigh := wilsonInterval(a.passed, a.verified, wilsonZ95)

	var costNorm, durNorm, tokenNorm, medCost, medDur, medTokens float64
	if a.evalCount > 0 {
		costNorm = a.costNormSum / float64(a.evalCount)
		durNorm = a.durNormSum / float64(a.evalCount)
		tokenNorm = a.tokenNormSum / float64(a.evalCount)
		medCost = a.costMedSum / float64(a.evalCount)
		medDur = a.durMedSum / float64(a.evalCount)
		medTokens = a.tokenMedSum / float64(a.evalCount)
	}

	qualScore := 0.0
	if a.qualCount > 0 {
		qualScore = a.qualSum / float64(a.qualCount)
	}

	return VariantRank{
		Name:              name,
		Runner:            a.runner,
		Model:             a.model,
		PassRate:          passRate,
		PassLow:           passLow,
		PassHigh:          passHigh,
		MedianCostUSD:     medCost,
		MedianDuration:    int64(medDur),
		MedianTotalTokens: int64(medTokens),
		QualityScore:      qualScore,
		CompositeScore: w.Correctness*passRate + w.Cost*costNorm +
			w.Duration*durNorm + w.Quality*qualScore + w.Tokens*tokenNorm,
		Tools: a.sortedTools(),
	}
}

// accumulate walks every eval, scoring cost/duration/tokens relative to the
// eval's own best variant, and folds the results into one accumulator per
// variant.
func accumulate(sr *result.SuiteResult) map[string]*variantAccumulator {
	accs := make(map[string]*variantAccumulator)
	for _, eval := range sr.Evals {
		scoreEval(eval, accs)
	}
	return accs
}

// evalMetric is a variant's within-eval median cost, duration, and total
// tokens (input + output). tokens is 0 when the variant reported no usage.
type evalMetric struct {
	name   string
	cost   float64
	dur    float64
	tokens float64
}

// scoreEval accumulates pass counts for every variant in the eval and adds each
// variant's cost/duration/tokens scored relative to the best variant in this eval.
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
			a.addToolCounts(run.ToolCounts)
		}

		agg := result.ComputeAggregate(v.Runs)
		if agg == nil {
			continue
		}
		metrics = append(metrics, evalMetric{
			name:   v.Name,
			cost:   agg.MedianCostUSD,
			dur:    float64(agg.MedianDurationMs),
			tokens: totalTokens(agg.Usage),
		})
	}

	if len(metrics) == 0 {
		return
	}

	foldMetrics(metrics, accs)
	scoreComparison(eval, accs)
}

// foldMetrics scores each variant's cost/duration/tokens relative to the eval's
// best variant and folds the results into the accumulators.
func foldMetrics(metrics []evalMetric, accs map[string]*variantAccumulator) {
	bestCost, bestDur, bestTokens := metrics[0].cost, metrics[0].dur, metrics[0].tokens
	for _, m := range metrics[1:] {
		if m.cost < bestCost {
			bestCost = m.cost
		}
		if m.dur < bestDur {
			bestDur = m.dur
		}
		if m.tokens < bestTokens {
			bestTokens = m.tokens
		}
	}

	for _, m := range metrics {
		a := accs[m.name]
		a.costNormSum += ratioLowerBetter(m.cost, bestCost)
		a.durNormSum += ratioLowerBetter(m.dur, bestDur)
		a.tokenNormSum += ratioLowerBetter(m.tokens, bestTokens)
		a.costMedSum += m.cost
		a.durMedSum += m.dur
		a.tokenMedSum += m.tokens
		a.evalCount++
	}
}

// totalTokens returns median total tokens (input + output) for a variant's
// aggregate, or 0 when no run reported usage. Cache tokens are excluded: the
// token dimension measures the model-agnostic input+output work, mirroring the
// "total tokens (input + output)" the reports surface.
func totalTokens(u *result.UsageAggregate) float64 {
	if u == nil {
		return 0
	}
	return float64(u.MedianInputTokens + u.MedianOutputTokens)
}

// scoreComparison folds this eval's comparative quality scores into each
// compared variant's accumulator. A skipped or absent comparison contributes
// nothing, leaving quality at 0 for variants never compared.
func scoreComparison(eval result.EvalResult, accs map[string]*variantAccumulator) {
	if eval.Comparison == nil || eval.Comparison.Skipped != "" {
		return
	}
	for _, s := range eval.Comparison.Scores {
		if a := accs[s.Variant]; a != nil {
			a.qualSum += s.Score
			a.qualCount++
		}
	}
}

// ratioLowerBetter scores a value against the best (lowest) value among the
// variants in an eval, for metrics where lower is better (cost, duration,
// tokens). The best value scores 1.0; a value twice the best scores 0.5, giving
// the composite sensitivity to the magnitude of the gap rather than only the
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
