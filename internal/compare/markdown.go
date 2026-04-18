package compare

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// WriteMarkdown writes a human-readable markdown comparison report to w.
func WriteMarkdown(w io.Writer, c *Comparison) {
	fmt.Fprintf(w, "# Comparison Report\n\n")
	fmt.Fprintf(w, "**Baseline:** %s\n", c.Baseline.StartedAt)
	fmt.Fprintf(w, "**Candidate:** %s\n\n", c.Candidate.StartedAt)

	for _, eval := range c.Evals {
		writeEvalComparison(w, eval)
	}
}

func writeEvalComparison(w io.Writer, eval EvalComparison) {
	name := eval.EvalName
	if name == "" {
		name = eval.EvalID
	}
	fmt.Fprintf(w, "## %s\n\n", name)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "TREATMENT\tSTATUS\tPASS RATE\tMEDIAN COST\tMEDIAN DURATION\n")
	fmt.Fprintf(tw, "---------\t------\t---------\t-----------\t---------------\n")

	for _, t := range eval.Treatments {
		switch t.Status {
		case StatusAdded:
			fmt.Fprintf(tw, "%s\tadded\t—\t—\t—\n", t.Name)
		case StatusRemoved:
			fmt.Fprintf(tw, "%s\tremoved\t—\t—\t—\n", t.Name)
		case StatusMatched:
			writeMatchedRow(tw, t)
		}
	}

	tw.Flush()
	fmt.Fprintln(w)
}

func writeMatchedRow(tw *tabwriter.Writer, t TreatmentComparison) {
	passCol := "—"
	if t.PassRateDelta != nil {
		passCol = fmt.Sprintf("%s%.0fpp %s",
			signPrefix(*t.PassRateDelta*100),
			abs(*t.PassRateDelta*100),
			direction(*t.PassRateDelta, true))
		if t.CandidatePassRate != nil {
			passCol += fmt.Sprintf(" (%.0f%%)", *t.CandidatePassRate*100)
		}
	} else if t.CandidatePassRate != nil {
		passCol = fmt.Sprintf("%.0f%%", *t.CandidatePassRate*100)
	}

	costCol := "—"
	if t.CostDelta != nil {
		costCol = fmt.Sprintf("%s$%.4f", signPrefix(*t.CostDelta), abs(*t.CostDelta))
		if t.CostDeltaPct != nil {
			costCol += fmt.Sprintf(" (%s%.1f%%)", signPrefix(*t.CostDeltaPct), abs(*t.CostDeltaPct))
		}
		costCol += " " + direction(*t.CostDelta, false)
	}

	durCol := "—"
	if t.DurationDeltaMs != nil {
		durCol = fmt.Sprintf("%s%s", signPrefix(float64(*t.DurationDeltaMs)), formatDuration(absInt64(*t.DurationDeltaMs)))
		if t.DurationDeltaPct != nil {
			durCol += fmt.Sprintf(" (%s%.1f%%)", signPrefix(*t.DurationDeltaPct), abs(*t.DurationDeltaPct))
		}
		durCol += " " + direction(float64(*t.DurationDeltaMs), false)
	}

	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", t.Name, "matched", passCol, costCol, durCol)
}

// direction returns an indicator: ↑ for increase, ↓ for decrease, = for no change.
// higherIsBetter inverts the semantics for pass rate vs cost/duration.
func direction(delta float64, higherIsBetter bool) string {
	if delta == 0 {
		return "="
	}
	if delta > 0 {
		if higherIsBetter {
			return "↑"
		}
		return "↑"
	}
	if higherIsBetter {
		return "↓"
	}
	return "↓"
}

func signPrefix(v float64) string {
	if v >= 0 {
		return "+"
	}
	return ""
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
