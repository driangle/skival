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
	fmt.Fprintf(w, "# Eval Report\n\n")
	if sr.Description != "" {
		fmt.Fprintf(w, "%s\n\n", sr.Description)
	}
	fmt.Fprintf(w, "**Started:** %s  \n", sr.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Finished:** %s  \n\n", sr.FinishedAt.Format("2006-01-02 15:04:05"))

	multi := hasMultipleRunners(sr)
	multiModel := hasMultipleModels(sr)
	writeResultsTable(w, sr, multi, multiModel)
	writeErrorsSection(w, sr)
	writeSkippedSection(w, sr)
	writeRankingTable(w, sr, multi, multiModel, weights)
}

// hasMultipleRunners returns true when the suite contains more than one distinct runner name.
func hasMultipleRunners(sr *result.SuiteResult) bool {
	seen := ""
	for _, eval := range sr.Evals {
		for _, treat := range eval.Treatments {
			if seen == "" {
				seen = treat.Runner
			} else if treat.Runner != seen {
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
		for _, treat := range eval.Treatments {
			if seen == "" {
				seen = treat.Model
			} else if treat.Model != seen {
				return true
			}
		}
	}
	return false
}

func writeResultsTable(w io.Writer, sr *result.SuiteResult, multiRunner, multiModel bool) {
	fmt.Fprintf(w, "## Results\n\n")

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "EVAL\tTREATMENT\tSAMPLE\tSTATUS\tCOST\tDURATION\n")
	fmt.Fprintf(tw, "----\t---------\t------\t------\t----\t--------\n")

	for _, eval := range sr.Evals {
		if eval.Err != nil {
			fmt.Fprintf(tw, "%s\t—\t—\tERROR\t—\t—\n", eval.EvalName)
			continue
		}
		for _, treat := range eval.Treatments {
			treatLabel := treatmentLabel(treat, multiRunner, multiModel)
			for _, run := range treat.Runs {
				status := runStatus(run)
				cost := fmt.Sprintf("$%.4f", run.CostUSD)
				duration := formatDuration(run.DurationMs)

				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
					eval.EvalName, treatLabel, run.Sample, status, cost, duration)
			}

			if agg := treat.Aggregate; agg != nil && len(treat.Runs) >= 2 {
				writeAggregateRow(tw, eval.EvalName, treatLabel, agg)
			}
		}
	}

	tw.Flush()
	fmt.Fprintln(w)
}

func treatmentLabel(t result.TreatmentResult, multiRunner, multiModel bool) string {
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

func writeAggregateRow(tw *tabwriter.Writer, evalName, treatName string, agg *result.Aggregate) {
	costRange := fmt.Sprintf("$%.4f [$%.4f\u2013$%.4f]", agg.MedianCostUSD, agg.MinCostUSD, agg.MaxCostUSD)
	durationRange := fmt.Sprintf("%s [%s\u2013%s]", formatDuration(agg.MedianDurationMs), formatDuration(agg.MinDurationMs), formatDuration(agg.MaxDurationMs))

	passStr := "\u2014"
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

	fmt.Fprintf(tw, "%s\t%s\tagg\t%s\t%s\t%s%s\n",
		evalName, treatName, passStr, costRange, durationRange, cvInfo)
}

func writeErrorsSection(w io.Writer, sr *result.SuiteResult) {
	var errors []result.EvalResult
	for _, eval := range sr.Evals {
		if eval.Err != nil {
			errors = append(errors, eval)
		}
	}
	if len(errors) == 0 {
		return
	}

	fmt.Fprintf(w, "## Errors\n\n")
	for _, eval := range errors {
		fmt.Fprintf(w, "- **%s** (`%s`): %v\n", eval.EvalName, eval.EvalID, eval.Err)
	}
	fmt.Fprintln(w)
}

func writeSkippedSection(w io.Writer, sr *result.SuiteResult) {
	var hasSkipped bool
	for _, eval := range sr.Evals {
		if len(eval.Skipped) > 0 {
			hasSkipped = true
			break
		}
	}
	if !hasSkipped {
		return
	}

	fmt.Fprintf(w, "## Skipped Treatments\n\n")
	for _, eval := range sr.Evals {
		if len(eval.Skipped) == 0 {
			continue
		}
		fmt.Fprintf(w, "**%s** (`%s`):\n", eval.EvalName, eval.EvalID)
		for _, s := range eval.Skipped {
			fmt.Fprintf(w, "- %s — %s\n", s.Name, s.Reason)
		}
	}
	fmt.Fprintln(w)
}

func writeRankingTable(w io.Writer, sr *result.SuiteResult, multiRunner, multiModel bool, weights Weights) {
	ranks := RankTreatments(sr, weights)
	if len(ranks) < 2 {
		return
	}

	fmt.Fprintf(w, "## Rankings\n\n")

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Build header dynamically based on which extra columns are needed.
	header := "RANK\tTREATMENT"
	sep := "----\t---------"
	if multiRunner {
		header += "\tRUNNER"
		sep += "\t------"
	}
	if multiModel {
		header += "\tMODEL"
		sep += "\t-----"
	}
	header += "\tSCORE\tPASS RATE\tMEDIAN COST\tMEDIAN DURATION\n"
	sep += "\t-----\t---------\t-----------\t---------------\n"
	fmt.Fprint(tw, header)
	fmt.Fprint(tw, sep)

	for _, r := range ranks {
		fmt.Fprintf(tw, "#%d\t%s", r.Rank, r.Name)
		if multiRunner {
			fmt.Fprintf(tw, "\t%s", r.Runner)
		}
		if multiModel {
			fmt.Fprintf(tw, "\t%s", r.Model)
		}
		fmt.Fprintf(tw, "\t%.3f\t%.0f%%\t$%.4f\t%s\n",
			r.CompositeScore,
			r.PassRate*100, r.MedianCostUSD,
			formatDuration(r.MedianDuration))
	}

	tw.Flush()
	fmt.Fprintln(w)
}

func runStatus(run result.RunResult) string {
	if run.Err != nil {
		return "error"
	}
	if run.Pass != nil {
		if *run.Pass {
			return "pass"
		}
		return "fail"
	}
	if run.IsError {
		return "failed"
	}
	return "ok"
}

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
