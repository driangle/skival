package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/driangle/skival/internal/result"
)

// jsonReport is the top-level JSON output structure.
type jsonReport struct {
	Description string          `json:"description"`
	StartedAt   string          `json:"started_at"`
	FinishedAt  string          `json:"finished_at"`
	Evals       []jsonEval      `json:"evals"`
	Rankings    []jsonRanking   `json:"rankings,omitempty"`
}

type jsonSkipped struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type jsonEval struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Error      string          `json:"error,omitempty"`
	Variants []jsonVariant `json:"variants"`
	Skipped    []jsonSkipped   `json:"skipped,omitempty"`
}

type jsonVariant struct {
	Name      string         `json:"name"`
	Runner    string         `json:"runner,omitempty"`
	Model     string         `json:"model,omitempty"`
	IsControl bool           `json:"is_control"`
	Runs      []jsonRun      `json:"runs"`
	Aggregate *jsonAggregate `json:"aggregate,omitempty"`
}

type jsonRun struct {
	Sample     int     `json:"sample"`
	Status     string  `json:"status"`
	CostUSD    float64 `json:"cost_usd"`
	DurationMs int64   `json:"duration_ms"`
	Pass       *bool   `json:"pass"`
	Error      string  `json:"error,omitempty"`
}

type jsonAggregate struct {
	MedianCostUSD    float64  `json:"median_cost_usd"`
	MinCostUSD       float64  `json:"min_cost_usd"`
	MaxCostUSD       float64  `json:"max_cost_usd"`
	MedianDurationMs int64    `json:"median_duration_ms"`
	MinDurationMs    int64    `json:"min_duration_ms"`
	MaxDurationMs    int64    `json:"max_duration_ms"`
	CostCV           *float64 `json:"cost_cv,omitempty"`
	DurationCV       *float64 `json:"duration_cv,omitempty"`
	Pass             *bool    `json:"pass"`
}

type jsonRanking struct {
	Rank           int     `json:"rank"`
	Name           string  `json:"name"`
	Runner         string  `json:"runner,omitempty"`
	Model          string  `json:"model,omitempty"`
	CompositeScore float64 `json:"composite_score"`
	PassRate       float64 `json:"pass_rate"`
	MedianCostUSD  float64 `json:"median_cost_usd"`
	MedianDuration int64   `json:"median_duration_ms"`
}

// WriteJSON writes a machine-readable JSON report to w.
func WriteJSON(w io.Writer, sr *result.SuiteResult, weights Weights) error {
	report := buildJSONReport(sr, weights)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encoding JSON report: %w", err)
	}
	return nil
}

func buildJSONReport(sr *result.SuiteResult, weights Weights) jsonReport {
	r := jsonReport{
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		FinishedAt:  sr.FinishedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	for _, eval := range sr.Evals {
		je := jsonEval{ID: eval.EvalID, Name: eval.EvalName}
		if eval.Err != nil {
			je.Error = eval.Err.Error()
		}
		for _, s := range eval.Skipped {
			je.Skipped = append(je.Skipped, jsonSkipped{Name: s.Name, Reason: s.Reason})
		}
		for _, v := range eval.Variants {
			jt := jsonVariant{
				Name:      v.Name,
				Runner:    v.Runner,
				Model:     v.Model,
				IsControl: v.IsControl,
			}
			for _, run := range v.Runs {
				jr := jsonRun{
					Sample:     run.Sample,
					Status:     runStatus(run),
					CostUSD:    run.CostUSD,
					DurationMs: run.DurationMs,
					Pass:       run.Pass,
				}
				if run.Err != nil {
					jr.Error = run.Err.Error()
				}
				jt.Runs = append(jt.Runs, jr)
			}
			if agg := v.Aggregate; agg != nil {
				jt.Aggregate = &jsonAggregate{
					MedianCostUSD:    agg.MedianCostUSD,
					MinCostUSD:       agg.MinCostUSD,
					MaxCostUSD:       agg.MaxCostUSD,
					MedianDurationMs: agg.MedianDurationMs,
					MinDurationMs:    agg.MinDurationMs,
					MaxDurationMs:    agg.MaxDurationMs,
					CostCV:           agg.CostCV,
					DurationCV:       agg.DurationCV,
					Pass:             agg.Pass,
				}
			}
			je.Variants = append(je.Variants, jt)
		}
		r.Evals = append(r.Evals, je)
	}

	ranks := RankVariants(sr, weights)
	for _, rank := range ranks {
		r.Rankings = append(r.Rankings, jsonRanking{
			Rank:           rank.Rank,
			Name:           rank.Name,
			Runner:         rank.Runner,
			Model:          rank.Model,
			CompositeScore: rank.CompositeScore,
			PassRate:       rank.PassRate,
			MedianCostUSD:  rank.MedianCostUSD,
			MedianDuration: rank.MedianDuration,
		})
	}

	return r
}
