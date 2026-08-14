package executor

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/driangle/skival/internal/color"
)

// WritePlan renders a dry-run run matrix to w: one row per eval×variant cell
// with its resolved runner, model, and sample count, followed by a summary of
// total cells and samples. When the plan has been priced (ApplyEstimate ran
// against a results dir), an EST COST column and an estimated total are shown.
func WritePlan(w io.Writer, plan *Plan) {
	priced := plan.EstimatedCells > 0
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	writePlanHeader(tw, priced)
	for i := range plan.Entries {
		writePlanRow(tw, &plan.Entries[i], priced)
	}
	tw.Flush()

	writePlanSummary(w, plan, priced)
}

func writePlanHeader(tw *tabwriter.Writer, priced bool) {
	if priced {
		fmt.Fprintf(tw, "EVAL\tVARIANT\tRUNNER\tMODEL\tSAMPLES\tEST COST\n")
		return
	}
	fmt.Fprintf(tw, "EVAL\tVARIANT\tRUNNER\tMODEL\tSAMPLES\n")
}

func writePlanRow(tw *tabwriter.Writer, e *PlanEntry, priced bool) {
	model := e.Model
	if model == "" {
		model = color.Dim("(default)")
	}
	runner := e.Runner
	if runner == "" {
		runner = color.Dim("(unset)")
	}
	if !priced {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n",
			color.Cyan(e.EvalName), color.Cyan(e.Variant), runner, model, e.Samples)
		return
	}
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
		color.Cyan(e.EvalName), color.Cyan(e.Variant), runner, model, e.Samples, planCost(e.EstCostUSD))
}

// planCost formats a cell's estimate, or a dim dash when it has no prior.
func planCost(cost *float64) string {
	if cost == nil {
		return color.Dim("—")
	}
	return color.Dimf("$%.4f", *cost)
}

func writePlanSummary(w io.Writer, plan *Plan, priced bool) {
	fmt.Fprintf(w, "\n%d evals×variants, %d total samples (dry run — nothing executed)\n",
		plan.TotalCells, plan.TotalSamples)
	if !priced {
		return
	}
	if plan.EstTotalUSD != nil {
		fmt.Fprintf(w, "Estimated total cost: $%.4f\n", *plan.EstTotalUSD)
	}
	if plan.EstimatedCells < plan.TotalCells {
		fmt.Fprintf(w, "%s\n", color.Dimf("(estimate covers %d of %d cells; the rest had no prior results)",
			plan.EstimatedCells, plan.TotalCells))
	}
}
