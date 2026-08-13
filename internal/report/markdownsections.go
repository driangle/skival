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
	fmt.Fprintf(w, "%s\n\n", artifactPointer(sr, "workdirs"))
}

// artifactPointer returns a one-line pointer to where per-sample artifacts of
// the given kind (e.g. "workdirs", "sessions") are recorded. When the suite was
// persisted it points at the results dir; otherwise it notes the paths were not
// saved to disk.
func artifactPointer(sr *result.SuiteResult, kind string) string {
	if sr.ResultsDir != "" {
		return fmt.Sprintf("Per-sample %s are recorded under `%s`.", kind, sr.ResultsDir)
	}
	return fmt.Sprintf("Per-sample %s were not persisted (run with `--results-dir` to save them).", kind)
}

// writeSessionsSection lists each run's agent session: a link to its static
// vibeview page when one was produced, otherwise a `vibeview show <id>` hint.
func writeSessionsSection(w io.Writer, sr *result.SuiteResult) {
	hasSession := false
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				if run.SessionID != "" {
					hasSession = true
				}
			}
		}
	}
	if !hasSession {
		return
	}

	fmt.Fprintf(w, "## Sessions\n\n")
	fmt.Fprintf(w, "%s\n\n", artifactPointer(sr, "sessions"))
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
	showTokens := weights.Tokens > 0

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	cols := rankingCols{multiRunner: multiRunner, multiModel: multiModel, quality: showQuality, tokens: showTokens}
	header, sep := rankingHeader(cols)
	fmt.Fprint(tw, header)
	fmt.Fprint(tw, sep)

	for _, r := range ranks {
		writeRankingRow(tw, r, cols)
	}

	tw.Flush()
	writeSignificanceNote(w, ranks)
	fmt.Fprintln(w)
}

// writeSignificanceNote warns when the top two variants' 95% Wilson pass-rate
// intervals overlap, meaning the #1 vs #2 gap is not distinguishable at this
// sample size. It is a no-op with fewer than two ranks or non-overlapping
// intervals.
func writeSignificanceNote(w io.Writer, ranks []VariantRank) {
	if len(ranks) < 2 {
		return
	}
	a, b := ranks[0], ranks[1]
	if !intervalsOverlap(a.PassLow, a.PassHigh, b.PassLow, b.PassHigh) {
		return
	}
	fmt.Fprintf(w, "\n> ⚠ #1 (%s) vs #2 (%s): not significant at this sample "+
		"size (pass-rate intervals overlap).\n", a.Name, b.Name)
}

// formatPassCI renders a variant's 95% Wilson pass-rate interval as "[21–83%]",
// rounding each bound to a whole percent.
func formatPassCI(r VariantRank) string {
	return fmt.Sprintf("[%.0f–%.0f%%]", r.PassLow*100, r.PassHigh*100)
}

// rankingCols selects which optional columns the ranking table shows.
type rankingCols struct {
	multiRunner bool
	multiModel  bool
	quality     bool
	tokens      bool
}

// rankingHeader builds the ranking table header and separator rows, including
// only the extra columns requested by the caller.
func rankingHeader(c rankingCols) (header, sep string) {
	header = "RANK\tVARIANT"
	sep = "----\t---------"
	if c.multiRunner {
		header += "\tRUNNER"
		sep += "\t------"
	}
	if c.multiModel {
		header += "\tMODEL"
		sep += "\t-----"
	}
	header += "\tSCORE\tPASS RATE\t95% CI"
	sep += "\t-----\t---------\t------"
	if c.quality {
		header += "\tQUALITY"
		sep += "\t-------"
	}
	header += "\tMEDIAN COST"
	sep += "\t-----------"
	if c.tokens {
		header += "\tMEDIAN TOKENS"
		sep += "\t-------------"
	}
	header += "\tMEDIAN DURATION\n"
	sep += "\t---------------\n"
	return header, sep
}

func writeRankingRow(tw *tabwriter.Writer, r VariantRank, c rankingCols) {
	fmt.Fprintf(tw, "#%d\t%s", r.Rank, r.Name)
	if c.multiRunner {
		fmt.Fprintf(tw, "\t%s", r.Runner)
	}
	if c.multiModel {
		fmt.Fprintf(tw, "\t%s", r.Model)
	}
	fmt.Fprintf(tw, "\t%.3f\t%.0f%%\t%s", r.CompositeScore, r.PassRate*100, formatPassCI(r))
	if c.quality {
		fmt.Fprintf(tw, "\t%.2f", r.QualityScore)
	}
	fmt.Fprintf(tw, "\t$%.4f", r.MedianCostUSD)
	if c.tokens {
		fmt.Fprintf(tw, "\t%s", formatTokens(r.MedianTotalTokens))
	}
	fmt.Fprintf(tw, "\t%s\n", formatDuration(r.MedianDuration))
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

func formatCost(usd float64) string {
	return fmt.Sprintf("$%.4f", usd)
}

// formatTokens renders a token count compactly: raw below 1k, then "1.2k" /
// "3.4M" so a wide range of totals stays readable in a fixed column.
func formatTokens(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
}

// formatTokenPair renders input/output token counts as "in/out", or "—" when
// both are zero (e.g. a runner that reports no usage).
func formatTokenPair(in, out int64) string {
	if in == 0 && out == 0 {
		return "—"
	}
	return formatTokens(in) + "/" + formatTokens(out)
}
