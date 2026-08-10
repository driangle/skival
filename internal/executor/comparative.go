package executor

import (
	"context"
	"fmt"
	"log/slog"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/verifier"
)

// runComparison runs comparative judging across the variants of an eval that
// all passed their per-run verification, attaching per-variant quality scores
// to er. It runs only as a tiebreaker (at least two passing variants) and
// degrades gracefully: any problem is recorded on er.Comparison.Skipped and
// leaves per-run pass/fail untouched.
func runComparison(ctx context.Context, eval *suite.Eval, suiteCmp *suite.Compare, er *result.EvalResult, reg *registry.Registry, override *bool) {
	cmp := suite.ResolveCompare(suiteCmp, eval.Compare, override)
	if cmp == nil {
		return
	}

	candidates := eligibleForComparison(er.Variants)
	if len(candidates) < 2 {
		er.Comparison = &result.Comparison{
			Skipped: fmt.Sprintf("comparison skipped: %d of %d variant(s) passed, need at least 2", len(candidates), len(er.Variants)),
		}
		return
	}

	model := cmp.Model
	if model == "" {
		model = verifier.DefaultJudgeModel
	}

	runner, err := reg.Create(candidates[0].runner, nil)
	if err != nil {
		er.Comparison = &result.Comparison{
			Model:   model,
			Skipped: fmt.Sprintf("comparison skipped: could not create judge runner: %v", err),
		}
		return
	}

	res := runComparativeJudge(ctx, eval, cmp, runner, candidates)
	er.Comparison = buildComparison(eval.ID, model, res)
}

// runComparativeJudge builds the comparative input from the candidates and runs
// the judge, returning its raw result.
func runComparativeJudge(ctx context.Context, eval *suite.Eval, cmp *suite.Compare, runner agentrunner.Runner, candidates []comparisonCandidate) verifier.ComparativeResult {
	vars := make([]verifier.ComparativeVariant, len(candidates))
	for i, c := range candidates {
		vars[i] = verifier.ComparativeVariant{Name: c.name, Output: c.output}
	}

	judge := &verifier.ComparativeJudge{Runner: runner, Model: cmp.Model}
	return judge.Compare(ctx, verifier.ComparativeInput{
		EvalPrompt: comparisonPrompt(eval),
		Criteria:   cmp.Criteria,
		Variants:   vars,
		MaxChars:   cmp.EffectiveMaxChars(),
	})
}

// buildComparison converts a comparative judge result into a Comparison,
// degrading gracefully to a skip note when the judge failed.
func buildComparison(evalID, model string, res verifier.ComparativeResult) *result.Comparison {
	comp := &result.Comparison{Model: model, Conversation: res.Conversation}
	if res.Err != nil {
		slog.Debug("Comparative judge degraded to pass/fail", "eval", evalID, "err", res.Err)
		comp.Skipped = fmt.Sprintf("comparison skipped: %v", res.Err)
		return comp
	}

	for _, s := range res.Scores {
		comp.Scores = append(comp.Scores, result.ComparativeScore{
			Variant: s.Variant,
			Rating:  s.Rating,
			Score:   s.Score,
			Reason:  s.Reason,
		})
	}
	return comp
}

// comparisonCandidate is a variant eligible for comparison: it passed and has a
// representative output.
type comparisonCandidate struct {
	name   string
	runner string
	output string
}

// eligibleForComparison returns the variants that passed every verified run,
// each paired with its first passing run's output. Variants with no verified
// runs, or any failing run, are excluded so comparison stays correctness-first.
func eligibleForComparison(variants []result.VariantResult) []comparisonCandidate {
	var out []comparisonCandidate
	for _, v := range variants {
		if output, ok := passingOutput(v); ok {
			out = append(out, comparisonCandidate{name: v.Name, runner: v.Runner, output: output})
		}
	}
	return out
}

// passingOutput reports whether a variant passed (has at least one verified run
// and every verified run passed) and returns its first passing run's output.
func passingOutput(v result.VariantResult) (string, bool) {
	hasVerified := false
	firstPass := ""
	firstPassSet := false
	for _, r := range v.Runs {
		if r.Pass == nil {
			continue
		}
		hasVerified = true
		if !*r.Pass {
			return "", false
		}
		if !firstPassSet {
			firstPass = r.Text
			firstPassSet = true
		}
	}
	return firstPass, hasVerified
}

// comparisonPrompt returns the task prompt shown to the comparative judge. It
// prefers the eval prompt, falling back to the first variant's prompt when the
// eval defines none (variant-only prompts).
func comparisonPrompt(eval *suite.Eval) string {
	if eval.Prompt != "" {
		return eval.Prompt
	}
	for i := range eval.Variants {
		if eval.Variants[i].Prompt != "" {
			return eval.Variants[i].Prompt
		}
	}
	return ""
}
