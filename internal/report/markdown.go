package report

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/driangle/skival/internal/result"
)

// WriteMarkdown writes a human-readable markdown report to w.
func WriteMarkdown(w io.Writer, sr *result.SuiteResult, weights Weights) {
	title := sr.Title
	if title == "" {
		title = "Eval Report"
	}
	fmt.Fprintf(w, "# %s\n\n", title)
	if sr.Description != "" {
		fmt.Fprintf(w, "%s\n\n", sr.Description)
	}
	fmt.Fprintf(w, "**Started:** %s  \n", sr.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Finished:** %s  \n\n", sr.FinishedAt.Format("2006-01-02 15:04:05"))

	multi := hasMultipleRunners(sr)
	multiModel := hasMultipleModels(sr)
	writeResultsTable(w, sr, multi, multiModel)
	writeRankingTable(w, sr, multi, multiModel, weights)
	writeWorkdirsSection(w, sr)
	writeSessionsSection(w, sr)
	writeComparisonSection(w, sr)
	writeErrorsSection(w, sr)
	writeFailuresSection(w, sr)
	writeSkippedSection(w, sr)
	writeResultsFooter(w, sr)
}

// writeResultsFooter echoes the results-dir location at the bottom of the
// report so a reader who redirected stderr can still find the saved artifacts.
// It is omitted when results were not persisted to disk.
func writeResultsFooter(w io.Writer, sr *result.SuiteResult) {
	if sr.ResultsDir == "" {
		return
	}
	fmt.Fprintf(w, "---\n\n**Results saved to** `%s`\n", sr.ResultsDir)
}

// writeComparisonSection renders per-eval comparative quality scores, and notes
// evals where comparison was requested but skipped (degraded to pass/fail).
func writeComparisonSection(w io.Writer, sr *result.SuiteResult) {
	var hasAny bool
	for _, eval := range sr.Evals {
		if eval.Comparison != nil {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return
	}

	fmt.Fprintf(w, "## Comparative Quality\n\n")
	for _, eval := range sr.Evals {
		if eval.Comparison != nil {
			writeComparisonEval(w, eval)
		}
	}
}

// writeComparisonEval renders a single eval's comparative quality block: either
// the score table, or a note when no scores were produced.
func writeComparisonEval(w io.Writer, eval result.EvalResult) {
	c := eval.Comparison
	name := evalDisplayName(eval)
	if len(c.Scores) == 0 {
		reason := c.Skipped
		if reason == "" {
			reason = "no scores produced"
		}
		fmt.Fprintf(w, "**%s** — %s\n\n", name, reason)
		return
	}

	model := c.Model
	if model != "" {
		model = fmt.Sprintf(" (judge: %s)", model)
	}
	fmt.Fprintf(w, "**%s**%s\n\n", name, model)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "VARIANT\tRATING\tSCORE\tREASON\n")
	fmt.Fprintf(tw, "---------\t------\t-----\t------\n")
	for _, s := range c.Scores {
		fmt.Fprintf(tw, "%s\t%d/5\t%.2f\t%s\n", s.Variant, s.Rating, s.Score, s.Reason)
	}
	tw.Flush()
	fmt.Fprintln(w)
}

// hasMultipleRunners returns true when the suite contains more than one distinct runner name.
func hasMultipleRunners(sr *result.SuiteResult) bool {
	seen := ""
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			if seen == "" {
				seen = v.Runner
			} else if v.Runner != seen {
				return true
			}
		}
	}
	return false
}

// hasMultipleModels returns true when the suite contains more than one distinct model.
func hasMultipleModels(sr *result.SuiteResult) bool {
	seen := ""
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			if seen == "" {
				seen = v.Model
			} else if v.Model != seen {
				return true
			}
		}
	}
	return false
}

func writeResultsTable(w io.Writer, sr *result.SuiteResult, multiRunner, multiModel bool) {
	fmt.Fprintf(w, "## Results\n\n")

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "EVAL\tVARIANT\tSAMPLE\tSTATUS\tCOST\tDURATION\tTOKENS (IN/OUT)\n")
	fmt.Fprintf(tw, "----\t---------\t------\t------\t----\t--------\t---------------\n")

	for _, eval := range sr.Evals {
		name := evalDisplayName(eval)
		if eval.Err != nil {
			fmt.Fprintf(tw, "%s\t—\t—\tERROR\t—\t—\t—\n", name)
			continue
		}
		for _, v := range eval.Variants {
			label := variantLabel(v, multiRunner, multiModel)
			for _, run := range v.Runs {
				status := runStatus(run)
				cost := fmt.Sprintf("$%.4f", run.CostUSD)
				duration := formatDuration(run.DurationMs)
				tokens := formatTokenPair(int64(run.Usage.InputTokens), int64(run.Usage.OutputTokens))

				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
					name, label, run.Sample, status, cost, duration, tokens)
			}

			if agg := v.Aggregate; agg != nil && len(v.Runs) >= 2 {
				writeAggregateRow(tw, name, label, agg)
			}
		}
	}

	tw.Flush()
	fmt.Fprintln(w)
}

func variantLabel(t result.VariantResult, multiRunner, multiModel bool) string {
	var annotations []string
	if multiRunner && t.Runner != "" {
		annotations = append(annotations, t.Runner)
	}
	if multiModel && t.Model != "" {
		annotations = append(annotations, t.Model)
	}
	if len(annotations) > 0 {
		return fmt.Sprintf("%s (%s)", t.Name, strings.Join(annotations, ", "))
	}
	return t.Name
}

func writeAggregateRow(tw *tabwriter.Writer, evalName, variantName string, agg *result.Aggregate) {
	costRange := fmt.Sprintf("$%.4f [$%.4f–$%.4f]", agg.MedianCostUSD, agg.MinCostUSD, agg.MaxCostUSD)
	durationRange := fmt.Sprintf("%s [%s–%s]", formatDuration(agg.MedianDurationMs), formatDuration(agg.MinDurationMs), formatDuration(agg.MaxDurationMs))

	passStr := "—"
	if agg.Pass != nil {
		if *agg.Pass {
			passStr = "PASS"
		} else {
			passStr = "FAIL"
		}
	}

	var parts []string
	if agg.CostCV != nil {
		parts = append(parts, fmt.Sprintf("cost_cv=%.1f%%", *agg.CostCV*100))
	}
	if agg.DurationCV != nil {
		parts = append(parts, fmt.Sprintf("dur_cv=%.1f%%", *agg.DurationCV*100))
	}
	cvInfo := ""
	if len(parts) > 0 {
		cvInfo = " " + strings.Join(parts, " ")
	}

	fmt.Fprintf(tw, "%s\t%s\tagg\t%s\t%s\t%s%s\t%s\n",
		evalName, variantName, passStr, costRange, durationRange, cvInfo, aggTokens(agg))
}

// aggTokens renders a variant's median input/output tokens for the aggregate
// row, or "—" when no token usage was recorded.
func aggTokens(agg *result.Aggregate) string {
	if agg.Usage == nil {
		return "—"
	}
	return formatTokenPair(agg.Usage.MedianInputTokens, agg.Usage.MedianOutputTokens)
}
