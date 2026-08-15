package report

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/driangle/skival/internal/result"
)

// formatToolCensus renders a variant's tool census compactly as
// "Read ×12, Grep ×4, TaskCreate ×10" (tools already sorted by count desc).
// Returns "—" when the variant used no tools.
func formatToolCensus(tools []ToolCount) string {
	if len(tools) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(tools))
	for _, t := range tools {
		parts = append(parts, fmt.Sprintf("%s ×%d", t.Name, t.Count))
	}
	return strings.Join(parts, ", ")
}

// sumRunToolCounts totals each tool's invocations across a variant's runs.
// Returns nil when no run used any tool, so the JSON field is omitted.
func sumRunToolCounts(runs []result.RunResult) map[string]int {
	var total map[string]int
	for _, run := range runs {
		for name, n := range run.ToolCounts {
			if total == nil {
				total = make(map[string]int)
			}
			total[name] += n
		}
	}
	return total
}

// anyToolCensus reports whether any variant recorded tool usage, gating the
// census section so suites whose runners report no tool blocks look unchanged.
func anyToolCensus(ranks []VariantRank) bool {
	for _, r := range ranks {
		if len(r.Tools) > 0 {
			return true
		}
	}
	return false
}

// rankLabel annotates a ranked variant with its runner and/or model when the
// suite mixes them, mirroring variantLabel for the results table.
func rankLabel(r VariantRank, multiRunner, multiModel bool) string {
	var annotations []string
	if multiRunner && r.Runner != "" {
		annotations = append(annotations, r.Runner)
	}
	if multiModel && r.Model != "" {
		annotations = append(annotations, r.Model)
	}
	if len(annotations) > 0 {
		return fmt.Sprintf("%s (%s)", r.Name, strings.Join(annotations, ", "))
	}
	return r.Name
}

// buildHTMLToolCensus builds the HTML tool-usage rows from the ranked variants.
// Unlike the rankings block it is not gated on having two variants, so a
// single-variant suite still shows what its one variant used. Returns nil when
// no variant used any tools.
func buildHTMLToolCensus(ranks []VariantRank, multiRunner, multiModel bool) []htmlToolCensus {
	if !anyToolCensus(ranks) {
		return nil
	}
	rows := make([]htmlToolCensus, 0, len(ranks))
	for _, r := range ranks {
		rows = append(rows, htmlToolCensus{
			Variant: rankLabel(r, multiRunner, multiModel),
			Tools:   formatToolCensus(r.Tools),
		})
	}
	return rows
}

// writeToolCensusSection renders, per variant, every tool it invoked with a
// count — making a tool used outside a variant's intended set obvious without
// inspecting raw JSONL. It is a no-op when no variant used any tools.
func writeToolCensusSection(w io.Writer, sr *result.SuiteResult, multiRunner, multiModel bool, weights Weights) {
	ranks := RankVariants(sr, weights)
	if !anyToolCensus(ranks) {
		return
	}

	fmt.Fprintf(w, "## Tool Usage\n\n")
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "VARIANT\tTOOLS\n")
	fmt.Fprintf(tw, "---------\t-----\n")
	for _, r := range ranks {
		fmt.Fprintf(tw, "%s\t%s\n", rankLabel(r, multiRunner, multiModel), formatToolCensus(r.Tools))
	}
	tw.Flush()
	fmt.Fprintln(w)
}
