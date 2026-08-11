package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/driangle/skival/internal/result"
)

// WriteHTML writes a self-contained HTML report to w.
func WriteHTML(w io.Writer, sr *result.SuiteResult, weights Weights) error {
	if err := reportTmpl.Execute(w, buildHTMLData(sr, weights)); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}
	return nil
}

func buildHTMLData(sr *result.SuiteResult, weights Weights) htmlData {
	multi := hasMultipleRunners(sr)
	multiModel := hasMultipleModels(sr)

	d := htmlData{
		CSS:          reportCSS,
		JS:           reportJS,
		Description:  sr.Description,
		StartedAt:    sr.StartedAt.Format("2006-01-02 15:04:05"),
		FinishedAt:   sr.FinishedAt.Format("2006-01-02 15:04:05"),
		WallDuration: formatDuration(sr.FinishedAt.Sub(sr.StartedAt).Milliseconds()),
		Counts:       buildHTMLCounts(sr),
		WeightsNote:  weightsNote(weights),
		ShowQuality:  hasComparison(sr),
	}

	ranks := RankVariants(sr, weights)
	d.Rankings = buildHTMLRankings(ranks, d.ShowQuality)
	d.Verdict = buildHTMLVerdict(ranks, d.ShowQuality)
	d.Health = buildHTMLHealth(sr)
	d.VariantNames = htmlVariantNames(sr)
	d.Evals = buildHTMLEvals(sr, multi, multiModel)
	d.Errors = buildHTMLErrors(sr)

	return d
}

// buildHTMLCounts renders the header's scope line: how much work this run
// covered, so a reader knows whether a difference is worth believing.
func buildHTMLCounts(sr *result.SuiteResult) string {
	variants := map[string]struct{}{}
	runs := 0
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			variants[v.Name] = struct{}{}
			runs += len(v.Runs)
		}
	}
	return fmt.Sprintf("%s · %s · %s",
		plural(len(sr.Evals), "eval"),
		plural(len(variants), "variant"),
		plural(runs, "sample"))
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// weightsNote spells out the composite formula next to the rankings, so the
// ordering is auditable without opening the config.
func weightsNote(w Weights) string {
	parts := []string{
		fmt.Sprintf("%.2g pass", w.Correctness),
		fmt.Sprintf("%.2g cost", w.Cost),
		fmt.Sprintf("%.2g speed", w.Duration),
	}
	if w.Quality > 0 {
		parts = append([]string{fmt.Sprintf("%.2g quality", w.Quality)}, parts...)
	}
	return "composite = " + strings.Join(parts, " · ")
}

// htmlVariantNames lists variant names in first-seen order for the filter chips.
func htmlVariantNames(sr *result.SuiteResult) []string {
	seen := map[string]struct{}{}
	var names []string
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			if _, ok := seen[v.Name]; ok {
				continue
			}
			seen[v.Name] = struct{}{}
			names = append(names, v.Name)
		}
	}
	if len(names) < 2 {
		return nil
	}
	return names
}

func buildHTMLErrors(sr *result.SuiteResult) []htmlError {
	var errors []htmlError
	for _, eval := range sr.Evals {
		if eval.Err != nil {
			errors = append(errors, htmlError{
				Name:    evalDisplayName(eval),
				ID:      eval.EvalID,
				Message: eval.Err.Error(),
			})
		}
	}
	return errors
}
