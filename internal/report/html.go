package report

import (
	"fmt"
	"html/template"
	"io"

	"github.com/driangle/skival/internal/result"
)

// WriteHTML writes a self-contained HTML report to w.
func WriteHTML(w io.Writer, sr *result.SuiteResult, weights Weights) error {
	data := buildHTMLData(sr, weights)

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}
	return nil
}

func buildHTMLData(sr *result.SuiteResult, weights Weights) htmlData {
	multi := hasMultipleRunners(sr)
	multiModel := hasMultipleModels(sr)

	d := htmlData{
		CSS:         reportCSS,
		JS:          reportJS,
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format("2006-01-02 15:04:05"),
		FinishedAt:  sr.FinishedAt.Format("2006-01-02 15:04:05"),
		MultiRunner: multi,
		MultiModel:  multiModel,
	}

	d.Results = buildHTMLResults(sr, multi, multiModel)
	d.Errors = buildHTMLErrors(sr)
	d.Skipped = buildHTMLSkipped(sr)
	d.Comparisons = buildHTMLComparisons(sr)
	buildHTMLRankings(&d, sr, weights)

	return d
}

func buildHTMLResults(sr *result.SuiteResult, multi, multiModel bool) []htmlResultRow {
	var rows []htmlResultRow
	for _, eval := range sr.Evals {
		if eval.Err != nil {
			rows = append(rows, htmlResultRow{
				Eval:   eval.EvalName,
				Status: "ERROR",
			})
			continue
		}
		for _, v := range eval.Variants {
			label := variantLabel(v, multi, multiModel)
			for _, run := range v.Runs {
				rows = append(rows, htmlResultRow{
					Eval:     eval.EvalName,
					Variant:  label,
					Sample:   fmt.Sprintf("%d", run.Sample),
					Status:   runStatus(run),
					Cost:     fmt.Sprintf("$%.4f", run.CostUSD),
					Duration: formatDuration(run.DurationMs),
				})
			}
			if agg := v.Aggregate; agg != nil && len(v.Runs) >= 2 {
				rows = append(rows, buildHTMLAggRow(eval.EvalName, label, agg))
			}
		}
	}
	return rows
}

func buildHTMLErrors(sr *result.SuiteResult) []htmlError {
	var errors []htmlError
	for _, eval := range sr.Evals {
		if eval.Err != nil {
			errors = append(errors, htmlError{
				Name:    eval.EvalName,
				ID:      eval.EvalID,
				Message: eval.Err.Error(),
			})
		}
	}
	return errors
}

func buildHTMLSkipped(sr *result.SuiteResult) []htmlSkippedGroup {
	var skipped []htmlSkippedGroup
	for _, eval := range sr.Evals {
		if len(eval.Skipped) == 0 {
			continue
		}
		group := htmlSkippedGroup{Name: eval.EvalName, ID: eval.EvalID}
		for _, s := range eval.Skipped {
			group.Entries = append(group.Entries, htmlSkippedEntry{Name: s.Name, Reason: s.Reason})
		}
		skipped = append(skipped, group)
	}
	return skipped
}

func buildHTMLComparisons(sr *result.SuiteResult) []htmlComparison {
	var comparisons []htmlComparison
	for _, eval := range sr.Evals {
		c := eval.Comparison
		if c == nil {
			continue
		}
		hc := htmlComparison{Name: eval.EvalName, ID: eval.EvalID, Model: c.Model, Skipped: c.Skipped}
		for _, s := range c.Scores {
			hc.Scores = append(hc.Scores, htmlComparativeScore{
				Variant: s.Variant,
				Rating:  fmt.Sprintf("%d/5", s.Rating),
				Score:   fmt.Sprintf("%.2f", s.Score),
				Reason:  s.Reason,
			})
		}
		comparisons = append(comparisons, hc)
	}
	return comparisons
}

func buildHTMLRankings(d *htmlData, sr *result.SuiteResult, weights Weights) {
	d.ShowQuality = hasComparison(sr)
	ranks := RankVariants(sr, weights)
	if len(ranks) < 2 {
		return
	}
	d.ShowRankings = true
	for _, r := range ranks {
		hr := htmlRanking{
			Rank:           r.Rank,
			Name:           r.Name,
			Runner:         r.Runner,
			Model:          r.Model,
			CompositeScore: fmt.Sprintf("%.3f", r.CompositeScore),
			PassRate:       fmt.Sprintf("%.0f%%", r.PassRate*100),
			MedianCost:     fmt.Sprintf("$%.4f", r.MedianCostUSD),
			MedianDuration: formatDuration(r.MedianDuration),
		}
		if d.ShowQuality {
			hr.QualityScore = fmt.Sprintf("%.2f", r.QualityScore)
		}
		d.Rankings = append(d.Rankings, hr)
	}
}

func buildHTMLAggRow(evalName, variantName string, agg *result.Aggregate) htmlResultRow {
	costRange := fmt.Sprintf("$%.4f [$%.4f–$%.4f]", agg.MedianCostUSD, agg.MinCostUSD, agg.MaxCostUSD)
	durationRange := fmt.Sprintf("%s [%s–%s]", formatDuration(agg.MedianDurationMs), formatDuration(agg.MinDurationMs), formatDuration(agg.MaxDurationMs))

	status := "—"
	if agg.Pass != nil {
		if *agg.Pass {
			status = "PASS"
		} else {
			status = "FAIL"
		}
	}

	var cvInfo string
	if agg.CostCV != nil {
		cvInfo += fmt.Sprintf("cost_cv=%.1f%%", *agg.CostCV*100)
	}
	if agg.DurationCV != nil {
		if cvInfo != "" {
			cvInfo += " "
		}
		cvInfo += fmt.Sprintf("dur_cv=%.1f%%", *agg.DurationCV*100)
	}

	return htmlResultRow{
		Eval:     evalName,
		Variant:  variantName,
		Sample:   "agg",
		Status:   status,
		Cost:     costRange,
		Duration: durationRange,
		IsAgg:    true,
		CVInfo:   cvInfo,
	}
}
