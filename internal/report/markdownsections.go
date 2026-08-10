package report

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/driangle/skival/internal/result"
)

func writeWorkdirsSection(w io.Writer, sr *result.SuiteResult) {
	var hasWorkdir bool
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				if run.WorkDir != "" {
					hasWorkdir = true
					break
				}
			}
		}
	}
	if !hasWorkdir {
		return
	}

	fmt.Fprintf(w, "## Workdirs\n\n")
	for _, eval := range sr.Evals {
		name := evalDisplayName(eval)
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				if run.WorkDir != "" {
					fmt.Fprintf(w, "- **%s** > %s > sample %d: `%s`\n",
						name, v.Name, run.Sample, run.WorkDir)
				}
			}
		}
	}
	fmt.Fprintln(w)
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

	fmt.Fprintf(w, "## Skipped Variants\n\n")
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
	ranks := RankVariants(sr, weights)
	if len(ranks) < 2 {
		return
	}

	fmt.Fprintf(w, "## Rankings\n\n")

	showQuality := hasComparison(sr)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	header, sep := rankingHeader(multiRunner, multiModel, showQuality)
	fmt.Fprint(tw, header)
	fmt.Fprint(tw, sep)

	for _, r := range ranks {
		writeRankingRow(tw, r, multiRunner, multiModel, showQuality)
	}

	tw.Flush()
	fmt.Fprintln(w)
}

// rankingHeader builds the ranking table header and separator rows, including
// only the extra columns requested by the caller.
func rankingHeader(multiRunner, multiModel, showQuality bool) (header, sep string) {
	header = "RANK\tVARIANT"
	sep = "----\t---------"
	if multiRunner {
		header += "\tRUNNER"
		sep += "\t------"
	}
	if multiModel {
		header += "\tMODEL"
		sep += "\t-----"
	}
	header += "\tSCORE\tPASS RATE"
	sep += "\t-----\t---------"
	if showQuality {
		header += "\tQUALITY"
		sep += "\t-------"
	}
	header += "\tMEDIAN COST\tMEDIAN DURATION\n"
	sep += "\t-----------\t---------------\n"
	return header, sep
}

func writeRankingRow(tw *tabwriter.Writer, r VariantRank, multiRunner, multiModel, showQuality bool) {
	fmt.Fprintf(tw, "#%d\t%s", r.Rank, r.Name)
	if multiRunner {
		fmt.Fprintf(tw, "\t%s", r.Runner)
	}
	if multiModel {
		fmt.Fprintf(tw, "\t%s", r.Model)
	}
	fmt.Fprintf(tw, "\t%.3f\t%.0f%%", r.CompositeScore, r.PassRate*100)
	if showQuality {
		fmt.Fprintf(tw, "\t%.2f", r.QualityScore)
	}
	fmt.Fprintf(tw, "\t$%.4f\t%s\n",
		r.MedianCostUSD,
		formatDuration(r.MedianDuration))
}

// evalDisplayName returns the eval name, falling back to ID.
func evalDisplayName(eval result.EvalResult) string {
	if eval.EvalName != "" {
		return eval.EvalName
	}
	return eval.EvalID
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
