package executor

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/driangle/skival/internal/color"
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
				status := color.Green("ok")
				if run.Err != nil {
					status = color.Red("error")
				} else if run.IsError {
					status = color.Red("failed")
				}

				cost := color.Dimf("$%.4f", run.CostUSD)
				duration := color.Dim(formatDuration(run.DurationMs))

				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
					color.Cyan(eval.EvalName), color.Cyan(treat.Name), run.Sample, status, cost, duration)
			}

			if agg := treat.Aggregate; agg != nil && len(treat.Runs) >= 2 {
				printAggregate(tw, eval.EvalName, treat.Name, agg)
			}
		}
	}

	tw.Flush()
}

func printAggregate(tw *tabwriter.Writer, evalName, treatName string, agg *result.Aggregate) {
	costRange := color.Dimf("$%.4f [$%.4f–$%.4f]", agg.MedianCostUSD, agg.MinCostUSD, agg.MaxCostUSD)
	durationRange := color.Dim(fmt.Sprintf("%s [%s–%s]", formatDuration(agg.MedianDurationMs), formatDuration(agg.MinDurationMs), formatDuration(agg.MaxDurationMs)))

	passStr := "—"
	if agg.Pass != nil {
		if *agg.Pass {
			passStr = color.Green("PASS")
		} else {
			passStr = color.Red("FAIL")
		}
	}

	cvInfo := ""
	if agg.CostCV != nil {
		cvInfo += color.Dimf(" cost_cv=%.1f%%", *agg.CostCV*100)
	}
	if agg.DurationCV != nil {
		cvInfo += color.Dimf(" dur_cv=%.1f%%", *agg.DurationCV*100)
	}

	fmt.Fprintf(tw, "%s\t%s\tagg\t%s\t%s\t%s%s\n",
		color.Cyan(evalName), color.Cyan(treatName), passStr, costRange, durationRange, cvInfo)
}
