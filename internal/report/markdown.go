package report

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/driangle/skival/internal/result"
)

// WriteMarkdown writes a human-readable markdown report to w.
func WriteMarkdown(w io.Writer, sr *result.SuiteResult) {
	fmt.Fprintf(w, "# Eval Report\n\n")
	if sr.Description != "" {
		fmt.Fprintf(w, "%s\n\n", sr.Description)
	}
	fmt.Fprintf(w, "**Started:** %s  \n", sr.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Finished:** %s  \n\n", sr.FinishedAt.Format("2006-01-02 15:04:05"))

	writeResultsTable(w, sr)
	writeErrorsSection(w, sr)
	writeRankingTable(w, sr)
}

func writeResultsTable(w io.Writer, sr *result.SuiteResult) {
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
			for _, run := range treat.Runs {
				status := runStatus(run)
				cost := fmt.Sprintf("$%.4f", run.CostUSD)
				duration := formatDuration(run.DurationMs)

				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
					eval.EvalName, treat.Name, run.Sample, status, cost, duration)
			}

			if agg := treat.Aggregate; agg != nil && len(treat.Runs) >= 2 {
				writeAggregateRow(tw, eval.EvalName, treat.Name, agg)
			}
		}
	}

	tw.Flush()
	fmt.Fprintln(w)
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

func writeRankingTable(w io.Writer, sr *result.SuiteResult) {
	ranks := RankTreatments(sr)
	if len(ranks) < 2 {
		return
	}

	fmt.Fprintf(w, "## Rankings\n\n")

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "RANK\tTREATMENT\tSCORE\tPASS RATE\tMEDIAN COST\tMEDIAN DURATION\n")
	fmt.Fprintf(tw, "----\t---------\t-----\t---------\t-----------\t---------------\n")

	for _, r := range ranks {
		fmt.Fprintf(tw, "#%d\t%s\t%.3f\t%.0f%%\t$%.4f\t%s\n",
			r.Rank, r.Name, r.CompositeScore,
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
