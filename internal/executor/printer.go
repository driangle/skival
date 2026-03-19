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

	fmt.Fprintf(tw, "EVAL\tTREATMENT\tSAMPLE\tSTATUS\tCOST\tDURATION\n")

	for _, eval := range sr.Evals {
		for _, treat := range eval.Treatments {
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
		}
	}

	tw.Flush()
}
