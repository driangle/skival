package report

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/driangle/skival/internal/result"
)

// buildHTMLVerdict answers the question a reviewer opens the report to ask:
// which variant won, and is the margin big enough to act on. It is derived
// entirely from the same ranks the rankings table renders.
func buildHTMLVerdict(ranks []VariantRank, showQuality bool) htmlVerdict {
	if len(ranks) < 2 {
		return htmlVerdict{}
	}
	first, second := ranks[0], ranks[1]

	return htmlVerdict{
		Winner:      first.Name,
		Summary:     verdictSummary(first, second, showQuality),
		ScoreMargin: fmt.Sprintf("%+.3f", first.CompositeScore-second.CompositeScore),
		CostDelta:   pctDelta(first.MedianCostUSD, second.MedianCostUSD),
		SpeedDelta:  speedRatio(first.MedianDuration, second.MedianDuration),
	}
}

// verdictSummary states the win in words, naming only the dimensions that
// actually differ so the sentence never overstates the result.
func verdictSummary(first, second VariantRank, showQuality bool) string {
	var clauses []string

	switch {
	case first.PassRate > second.PassRate:
		clauses = append(clauses, "Passes more often than the runner-up")
	case first.PassRate == second.PassRate:
		clauses = append(clauses, "Matches the runner-up on pass rate")
	default:
		clauses = append(clauses, "Passes less often than the runner-up, but ranks ahead overall")
	}
	if showQuality && first.QualityScore > second.QualityScore {
		clauses = append(clauses, "is judged higher on quality")
	}
	clauses = append(clauses, costSpeedClause(first, second))

	return strings.Join(clauses, ", ") + "."
}

func costSpeedClause(first, second VariantRank) string {
	cheaper := first.MedianCostUSD < second.MedianCostUSD
	faster := first.MedianDuration < second.MedianDuration
	switch {
	case cheaper && faster:
		return "and costs less while finishing sooner"
	case cheaper:
		return "and costs less per run"
	case faster:
		return "and finishes sooner"
	default:
		return "and wins on score despite costing more time or money"
	}
}

// pctDelta renders first relative to second as a signed percentage, phrased so
// a negative number always means "the winner is cheaper".
func pctDelta(first, second float64) string {
	if second == 0 {
		return "—"
	}
	pct := (first - second) / second * 100
	return fmt.Sprintf("%+.0f%%", pct)
}

// speedRatio renders how many times faster (or slower) the winner is.
func speedRatio(first, second int64) string {
	if first <= 0 || second <= 0 {
		return "—"
	}
	if second >= first {
		return fmt.Sprintf("%.1f× faster", float64(second)/float64(first))
	}
	return fmt.Sprintf("%.1f× slower", float64(first)/float64(second))
}

// buildHTMLHealth summarizes correctness as one cell per sample run, in run
// order, so a failing sample is visible before any table is read.
func buildHTMLHealth(sr *result.SuiteResult) htmlHealth {
	var h htmlHealth
	var spend float64
	runners := map[string]struct{}{}
	var runnerList []string
	failures := 0

	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			if _, ok := runners[v.Runner]; !ok && v.Runner != "" {
				runners[v.Runner] = struct{}{}
				runnerList = append(runnerList, v.Runner)
			}
			for _, run := range v.Runs {
				spend += run.CostUSD
				ok := isPassing(run)
				if !ok {
					failures++
				}
				h.Cells = append(h.Cells, htmlHealthCell{Pass: ok, Title: cellTitle(eval, v, run)})
			}
		}
	}

	total := len(h.Cells)
	h.PassSummary = fmt.Sprintf("%d/%d", total-failures, total)
	h.TotalSpend = formatCost(spend)
	h.Runners = orDash(strings.Join(runnerList, ", "))
	h.Judge = judgeModel(sr)
	return h
}

func isPassing(run result.RunResult) bool {
	switch runStatus(run) {
	case "fail", "failed", "error":
		return false
	default:
		return true
	}
}

func cellTitle(eval result.EvalResult, v result.VariantResult, run result.RunResult) string {
	return fmt.Sprintf("%s · %s · sample %d — %s", evalDisplayName(eval), v.Name, run.Sample, runStatus(run))
}

func judgeModel(sr *result.SuiteResult) string {
	for _, eval := range sr.Evals {
		if eval.Comparison != nil && eval.Comparison.Model != "" {
			return eval.Comparison.Model
		}
	}
	return ""
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// buildHTMLRankings renders each rank as a row with bars scaled against the
// worst value in its column, so relative standing reads without arithmetic.
func buildHTMLRankings(ranks []VariantRank, showQuality bool) []htmlRanking {
	if len(ranks) < 2 {
		return nil
	}
	var maxScore, maxQual, maxCost float64
	var maxDur int64
	for _, r := range ranks {
		maxScore = max(maxScore, r.CompositeScore)
		maxQual = max(maxQual, qualityOrPass(r, showQuality))
		maxCost = max(maxCost, r.MedianCostUSD)
		maxDur = max(maxDur, r.MedianDuration)
	}

	rows := make([]htmlRanking, 0, len(ranks))
	for _, r := range ranks {
		rows = append(rows, htmlRanking{
			Rank:           r.Rank,
			Name:           r.Name,
			Attribution:    orDash(strings.Trim(strings.Join([]string{r.Runner, r.Model}, " · "), " ·")),
			CompositeScore: fmt.Sprintf("%.3f", r.CompositeScore),
			PassRate:       fmt.Sprintf("%.0f%%", r.PassRate*100),
			QualityScore:   fmt.Sprintf("%.2f", r.QualityScore),
			MedianCost:     formatCost(r.MedianCostUSD),
			MedianDuration: formatDuration(r.MedianDuration),
			CompositeWidth: widthCSS(r.CompositeScore, maxScore),
			QualityWidth:   widthCSS(qualityOrPass(r, showQuality), maxQual),
			CostWidth:      widthCSS(r.MedianCostUSD, maxCost),
			DurationWidth:  widthCSS(float64(r.MedianDuration), float64(maxDur)),
		})
	}
	return rows
}

func qualityOrPass(r VariantRank, showQuality bool) float64 {
	if showQuality {
		return r.QualityScore
	}
	return r.PassRate
}

// widthCSS is a bar width as a percentage of the column's largest value.
func widthCSS(val, maxVal float64) template.CSS {
	if maxVal <= 0 {
		return template.CSS("width:0%")
	}
	pct := max(0, min(100, val/maxVal*100))
	return template.CSS(fmt.Sprintf("width:%.1f%%", pct))
}
