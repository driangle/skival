package report

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/driangle/skival/internal/result"
)

// buildHTMLEvals renders one collapsible card per eval. The first card starts
// expanded so the report is useful without a click; the rest collapse to keep
// a long suite scannable.
func buildHTMLEvals(sr *result.SuiteResult, multi, multiModel bool) []htmlEval {
	evals := make([]htmlEval, 0, len(sr.Evals))
	for i, eval := range sr.Evals {
		e := htmlEval{
			ID:      eval.EvalID,
			Name:    evalDisplayName(eval),
			Open:    i == 0,
			Rows:    buildHTMLRows(eval, multi, multiModel),
			Skipped: buildHTMLSkipped(eval),
		}
		if eval.Err != nil {
			e.Note = "This eval did not run: " + eval.Err.Error()
		}
		if c := eval.Comparison; c != nil {
			e.Judge = c.Model
			e.Verdicts = buildHTMLVerdicts(c)
			if len(e.Verdicts) == 0 && e.Note == "" {
				e.Note = comparisonNote(c)
			}
		}
		e.Summary = evalSummary(eval)
		evals = append(evals, e)
	}
	return evals
}

func comparisonNote(c *result.Comparison) string {
	if c.Skipped != "" {
		return "Comparative judging skipped: " + c.Skipped
	}
	return "Comparative judging produced no scores."
}

// evalSummary is the one-line count shown on the collapsed card header.
func evalSummary(eval result.EvalResult) string {
	total, failures := 0, 0
	for _, v := range eval.Variants {
		for _, run := range v.Runs {
			total++
			if !isPassing(run) {
				failures++
			}
		}
	}
	if total == 0 {
		return "no samples"
	}
	return fmt.Sprintf("%d/%d passed · %s", total-failures, total, plural(len(eval.Variants), "variant"))
}

// buildHTMLRows emits every sample run plus, for multi-sample variants, an
// aggregate row carrying the median and the min–max spread.
func buildHTMLRows(eval result.EvalResult, multi, multiModel bool) []htmlResultRow {
	slowest := slowestRun(eval)
	var rows []htmlResultRow

	for _, v := range eval.Variants {
		label := variantLabel(v, multi, multiModel)
		for _, run := range v.Runs {
			status := runStatus(run)
			tokens, tokensTitle := runTokens(run)
			rows = append(rows, htmlResultRow{
				Variant:      label,
				Sample:       fmt.Sprintf("%d", run.Sample),
				Status:       status,
				StatusClass:  statusClass(status),
				Cost:         formatCost(run.CostUSD),
				Duration:     formatDuration(run.DurationMs),
				Tokens:       tokens,
				TokensTitle:  tokensTitle,
				SpanStyle:    spanCSS(0, run.DurationMs, slowest),
				Detail:       runErrorMessage(run),
				SessionPage:  run.SessionPage,
				SessionID:    run.SessionID,
				SessionShort: shortSessionID(run.SessionID),
			})
		}
		if agg := v.Aggregate; agg != nil && len(v.Runs) >= 2 {
			rows = append(rows, buildHTMLAggRow(label, agg, slowest))
		}
	}
	return rows
}

// buildHTMLAggRow is the emphasized per-variant row: median values, the
// min–max duration band, and the coefficient of variation when computable.
func buildHTMLAggRow(variant string, agg *result.Aggregate, slowest int64) htmlResultRow {
	status := "—"
	class := "na"
	if agg.Pass != nil {
		if *agg.Pass {
			status, class = "PASS", "pass"
		} else {
			status, class = "FAIL", "fail"
		}
	}
	tokens, tokensTitle := aggUsageTokens(agg.Usage)
	return htmlResultRow{
		Variant:     variant,
		Sample:      "median",
		Status:      status,
		StatusClass: class,
		Cost:        formatCost(agg.MedianCostUSD),
		Duration:    formatDuration(agg.MedianDurationMs),
		Tokens:      tokens,
		TokensTitle: tokensTitle,
		CVInfo:      cvInfo(agg),
		SpanStyle:   spanCSS(agg.MinDurationMs, agg.MaxDurationMs, slowest),
		IsAgg:       true,
	}
}

// runTokens returns a run's compact input/output token display and a full
// breakdown title for the hover tooltip. Both are empty when no usage exists.
func runTokens(run result.RunResult) (display, title string) {
	u := run.Usage
	if u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0 {
		return "", ""
	}
	display = formatTokenPair(int64(u.InputTokens), int64(u.OutputTokens))
	title = fmt.Sprintf("in %d · out %d · cache write %d · cache read %d",
		u.InputTokens, u.OutputTokens, u.CacheCreationInputTokens, u.CacheReadInputTokens)
	return display, title
}

// aggUsageTokens returns the median input/output token display and a breakdown
// title for the aggregate row. Both are empty when no usage exists.
func aggUsageTokens(u *result.UsageAggregate) (display, title string) {
	if u == nil {
		return "", ""
	}
	display = formatTokenPair(u.MedianInputTokens, u.MedianOutputTokens)
	title = fmt.Sprintf("median — in %d · out %d · cache write %d · cache read %d",
		u.MedianInputTokens, u.MedianOutputTokens, u.MedianCacheCreationTokens, u.MedianCacheReadTokens)
	return display, title
}

func cvInfo(agg *result.Aggregate) string {
	var parts []string
	if agg.CostCV != nil {
		parts = append(parts, fmt.Sprintf("cost cv %.0f%%", *agg.CostCV*100))
	}
	if agg.DurationCV != nil {
		parts = append(parts, fmt.Sprintf("dur cv %.0f%%", *agg.DurationCV*100))
	}
	return strings.Join(parts, " · ")
}

// slowestRun is the eval's longest single run, the scale every duration bar in
// this eval is drawn against.
func slowestRun(eval result.EvalResult) int64 {
	var slowest int64
	for _, v := range eval.Variants {
		for _, run := range v.Runs {
			slowest = max(slowest, run.DurationMs)
		}
	}
	return slowest
}

// spanCSS positions a duration band inside the eval's slowest run. A zero-width
// band is floored at 2% so a single-sample variant still renders a mark.
func spanCSS(from, to, scale int64) template.CSS {
	if scale <= 0 {
		return template.CSS("left:0%;width:0%")
	}
	left := float64(from) / float64(scale) * 100
	width := float64(to-from) / float64(scale) * 100
	if width < 2 {
		width = 2
	}
	if left+width > 100 {
		left = 100 - width
	}
	return template.CSS(fmt.Sprintf("left:%.1f%%;width:%.1f%%", left, width))
}

func statusClass(status string) string {
	switch status {
	case "pass":
		return "pass"
	case "fail", "failed":
		return "fail"
	case "error":
		return "error"
	default:
		return "na"
	}
}

// buildHTMLVerdicts turns judge scores into expandable rows: rating pips and a
// first-sentence teaser up front, full rationale on click.
func buildHTMLVerdicts(c *result.Comparison) []htmlJudgeVerdict {
	verdicts := make([]htmlJudgeVerdict, 0, len(c.Scores))
	for _, s := range c.Scores {
		pips := make([]bool, 5)
		for i := range pips {
			pips[i] = i < s.Rating
		}
		verdicts = append(verdicts, htmlJudgeVerdict{
			Variant: s.Variant,
			Rating:  fmt.Sprintf("%d/5", s.Rating),
			Score:   fmt.Sprintf("%.2f", s.Score),
			Teaser:  firstSentence(s.Reason),
			Reason:  s.Reason,
			Pips:    pips,
		})
	}
	return verdicts
}

// firstSentence is the collapsed preview of a judge rationale.
func firstSentence(reason string) string {
	reason = strings.TrimSpace(reason)
	if i := strings.Index(reason, ". "); i > 0 {
		return reason[:i+1]
	}
	return reason
}

// shortSessionID abbreviates a session id for compact display, keeping the
// leading segment that vibeview's prefix matching accepts.
func shortSessionID(id string) string {
	const n = 8
	if len(id) <= n {
		return id
	}
	return id[:n]
}

// runErrorMessage is the reason a run errored, or "" if it did not error.
func runErrorMessage(run result.RunResult) string {
	if run.Err != nil {
		return run.Err.Error()
	}
	if run.IsError {
		return "run errored without a message"
	}
	return ""
}

func buildHTMLSkipped(eval result.EvalResult) []htmlSkippedEntry {
	entries := make([]htmlSkippedEntry, 0, len(eval.Skipped))
	for _, s := range eval.Skipped {
		entries = append(entries, htmlSkippedEntry{Name: s.Name, Reason: "skipped — " + s.Reason})
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}
