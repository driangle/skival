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

type jsonEval struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Error      string          `json:"error,omitempty"`
	Treatments []jsonTreatment `json:"treatments"`
}

type jsonTreatment struct {
	Name      string         `json:"name"`
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
	CompositeScore float64 `json:"composite_score"`
	PassRate       float64 `json:"pass_rate"`
	MedianCostUSD  float64 `json:"median_cost_usd"`
	MedianDuration int64   `json:"median_duration_ms"`
}

// WriteJSON writes a machine-readable JSON report to w.
func WriteJSON(w io.Writer, sr *result.SuiteResult) error {
	report := buildJSONReport(sr)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encoding JSON report: %w", err)
	}
	return nil
}

func buildJSONReport(sr *result.SuiteResult) jsonReport {
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
		for _, treat := range eval.Treatments {
			jt := jsonTreatment{
				Name:      treat.Name,
				IsControl: treat.IsControl,
			}
			for _, run := range treat.Runs {
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
			if agg := treat.Aggregate; agg != nil {
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
			je.Treatments = append(je.Treatments, jt)
		}
		r.Evals = append(r.Evals, je)
	}

	ranks := RankTreatments(sr)
	for _, rank := range ranks {
		r.Rankings = append(r.Rankings, jsonRanking{
			Rank:           rank.Rank,
			Name:           rank.Name,
			CompositeScore: rank.CompositeScore,
			PassRate:       rank.PassRate,
			MedianCostUSD:  rank.MedianCostUSD,
			MedianDuration: rank.MedianDuration,
		})
	}

	return r
}
