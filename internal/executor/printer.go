package executor

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/driangle/skival/internal/result"
)

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// PrintResults writes a formatted table of suite results to w.
func PrintResults(w io.Writer, sr *result.SuiteResult) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprintf(tw, "EVAL\tVARIANT\tSAMPLE\tSTATUS\tCOST\tDURATION\n")

	for _, eval := range sr.Evals {
		for _, treat := range eval.Variants {
			for _, run := range treat.Runs {
				status := "ok"
				if run.Err != nil {
					status = "error"
				} else if run.IsError {
					status = "failed"
				}

				cost := fmt.Sprintf("$%.4f", run.CostUSD)
				duration := formatDuration(run.DurationMs)

				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
					eval.EvalName, treat.Name, run.Sample, status, cost, duration)
			}

			if agg := treat.Aggregate; agg != nil && len(treat.Runs) >= 2 {
				printAggregate(tw, eval.EvalName, treat.Name, agg)
			}
		}
	}

	tw.Flush()
}

func printAggregate(tw *tabwriter.Writer, evalName, treatName string, agg *result.Aggregate) {
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

	cvInfo := ""
	if agg.CostCV != nil {
		cvInfo += fmt.Sprintf(" cost_cv=%.1f%%", *agg.CostCV*100)
	}
	if agg.DurationCV != nil {
		cvInfo += fmt.Sprintf(" dur_cv=%.1f%%", *agg.DurationCV*100)
	}

	fmt.Fprintf(tw, "%s\t%s\tagg\t%s\t%s\t%s%s\n",
		evalName, treatName, passStr, costRange, durationRange, cvInfo)
}
