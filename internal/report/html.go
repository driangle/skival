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
		Title:        sr.Title,
		Description:  sr.Description,
		StartedAt:    sr.StartedAt.Format("2006-01-02 15:04:05"),
		FinishedAt:   sr.FinishedAt.Format("2006-01-02 15:04:05"),
		WallDuration: formatDuration(sr.FinishedAt.Sub(sr.StartedAt).Milliseconds()),
		Counts:       buildHTMLCounts(sr),
		WeightsNote:  weightsNote(weights),
		ShowQuality:  hasComparison(sr),
		ShowTokens:   weights.Tokens > 0,
	}

	ranks := RankVariants(sr, weights)
	d.Rankings = buildHTMLRankings(ranks, d.ShowQuality, d.ShowTokens)
	d.ToolCensus = buildHTMLToolCensus(ranks, multi, multiModel)
	d.Verdict = buildHTMLVerdict(ranks, d.ShowQuality)
	d.Health = buildHTMLHealth(sr)
	d.VariantNames = htmlVariantNames(sr)
	d.Evals = buildHTMLEvals(sr, multi, multiModel)
	d.Errors = buildHTMLErrors(sr)
	d.HasSessions = anySession(sr)
	d.HasTokens = anyTokens(sr)
	d.DetailColSpan = detailColSpan(d.HasTokens, d.HasSessions)

	return d
}

// detailColSpan counts the samples table's columns (6 fixed: variant, sample,
// status, cost, duration, spread) plus the optional Tokens and Session columns,
// so the expandable error-detail row spans the full table width.
func detailColSpan(hasTokens, hasSessions bool) int {
	span := 6
	if hasTokens {
		span++
	}
	if hasSessions {
		span++
	}
	return span
}

// anyTokens reports whether any run recorded token usage, gating the "Tokens"
// column so suites from runners without usage (e.g. exec) render unchanged.
func anyTokens(sr *result.SuiteResult) bool {
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				u := run.Usage
				if u.InputTokens != 0 || u.OutputTokens != 0 ||
					u.CacheCreationInputTokens != 0 || u.CacheReadInputTokens != 0 {
					return true
				}
			}
		}
	}
	return false
}

// anySession reports whether any run carries a session id, gating the "Session"
// column so suites without session data (e.g. exec-runner) look unchanged.
func anySession(sr *result.SuiteResult) bool {
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				if run.SessionID != "" {
					return true
				}
			}
		}
	}
	return false
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
	if w.Tokens > 0 {
		parts = append(parts, fmt.Sprintf("%.2g tokens", w.Tokens))
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
