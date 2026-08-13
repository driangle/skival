package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/driangle/skival/internal/result"
)

// jsonReport is the top-level JSON output structure.
type jsonReport struct {
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description"`
	StartedAt   string        `json:"started_at"`
	FinishedAt  string        `json:"finished_at"`
	Evals       []jsonEval    `json:"evals"`
	Rankings    []jsonRanking `json:"rankings,omitempty"`
}

type jsonSkipped struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type jsonEval struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Error      string          `json:"error,omitempty"`
	Variants   []jsonVariant   `json:"variants"`
	Skipped    []jsonSkipped   `json:"skipped,omitempty"`
	Comparison *jsonComparison `json:"comparison,omitempty"`
}

type jsonComparison struct {
	Model   string                 `json:"model,omitempty"`
	Skipped string                 `json:"skipped,omitempty"`
	Scores  []jsonComparativeScore `json:"scores,omitempty"`
}

type jsonComparativeScore struct {
	Variant string  `json:"variant"`
	Rating  int     `json:"rating"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason,omitempty"`
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
	Sample      int        `json:"sample"`
	Status      string     `json:"status"`
	CostUSD     float64    `json:"cost_usd"`
	DurationMs  int64      `json:"duration_ms"`
	Usage       *jsonUsage `json:"usage,omitempty"`
	Pass        *bool      `json:"pass"`
	Error       string     `json:"error,omitempty"`
	SessionID   string     `json:"session_id,omitempty"`
	SessionPage string     `json:"session_page,omitempty"`
}

// jsonUsage is the per-run token breakdown. Nil when the run reported no tokens.
type jsonUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
	CacheReadTokens     int `json:"cache_read_tokens"`
}

type jsonAggregate struct {
	MedianCostUSD    float64       `json:"median_cost_usd"`
	MinCostUSD       float64       `json:"min_cost_usd"`
	MaxCostUSD       float64       `json:"max_cost_usd"`
	MedianDurationMs int64         `json:"median_duration_ms"`
	MinDurationMs    int64         `json:"min_duration_ms"`
	MaxDurationMs    int64         `json:"max_duration_ms"`
	CostCV           *float64      `json:"cost_cv,omitempty"`
	DurationCV       *float64      `json:"duration_cv,omitempty"`
	Usage            *jsonUsageAgg `json:"usage,omitempty"`
	Pass             *bool         `json:"pass"`
}

// jsonUsageAgg is the per-variant median token usage. Nil when no tokens exist.
type jsonUsageAgg struct {
	MedianInputTokens         int64 `json:"median_input_tokens"`
	MedianOutputTokens        int64 `json:"median_output_tokens"`
	MedianCacheCreationTokens int64 `json:"median_cache_creation_tokens"`
	MedianCacheReadTokens     int64 `json:"median_cache_read_tokens"`
}

type jsonRanking struct {
	Rank           int      `json:"rank"`
	Name           string   `json:"name"`
	Runner         string   `json:"runner,omitempty"`
	Model          string   `json:"model,omitempty"`
	CompositeScore float64  `json:"composite_score"`
	PassRate       float64  `json:"pass_rate"`
	MedianCostUSD  float64  `json:"median_cost_usd"`
	MedianDuration int64    `json:"median_duration_ms"`
	QualityScore   *float64 `json:"quality_score,omitempty"`
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
		Title:       sr.Title,
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		FinishedAt:  sr.FinishedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	for _, eval := range sr.Evals {
		r.Evals = append(r.Evals, buildJSONEval(eval))
	}
	r.Rankings = buildJSONRankings(sr, weights)

	return r
}

func buildJSONEval(eval result.EvalResult) jsonEval {
	je := jsonEval{ID: eval.EvalID, Name: eval.EvalName}
	if eval.Err != nil {
		je.Error = eval.Err.Error()
	}
	for _, s := range eval.Skipped {
		je.Skipped = append(je.Skipped, jsonSkipped{Name: s.Name, Reason: s.Reason})
	}
	if c := eval.Comparison; c != nil {
		jc := &jsonComparison{Model: c.Model, Skipped: c.Skipped}
		for _, s := range c.Scores {
			jc.Scores = append(jc.Scores, jsonComparativeScore{
				Variant: s.Variant,
				Rating:  s.Rating,
				Score:   s.Score,
				Reason:  s.Reason,
			})
		}
		je.Comparison = jc
	}
	for _, v := range eval.Variants {
		je.Variants = append(je.Variants, buildJSONVariant(v))
	}
	return je
}

func buildJSONVariant(v result.VariantResult) jsonVariant {
	jt := jsonVariant{
		Name:      v.Name,
		Runner:    v.Runner,
		Model:     v.Model,
		IsControl: v.IsControl,
	}
	for _, run := range v.Runs {
		jr := jsonRun{
			Sample:      run.Sample,
			Status:      runStatus(run),
			CostUSD:     run.CostUSD,
			DurationMs:  run.DurationMs,
			Usage:       jsonRunUsage(run),
			Pass:        run.Pass,
			SessionID:   run.SessionID,
			SessionPage: run.SessionPage,
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
			Usage:            jsonAggUsage(agg.Usage),
			Pass:             agg.Pass,
		}
	}
	return jt
}

// jsonRunUsage renders a run's token usage, or nil when it reported none.
func jsonRunUsage(run result.RunResult) *jsonUsage {
	u := run.Usage
	if u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0 {
		return nil
	}
	return &jsonUsage{
		InputTokens:         u.InputTokens,
		OutputTokens:        u.OutputTokens,
		CacheCreationTokens: u.CacheCreationInputTokens,
		CacheReadTokens:     u.CacheReadInputTokens,
	}
}

// jsonAggUsage renders a variant's median token usage, or nil when absent.
func jsonAggUsage(u *result.UsageAggregate) *jsonUsageAgg {
	if u == nil {
		return nil
	}
	return &jsonUsageAgg{
		MedianInputTokens:         u.MedianInputTokens,
		MedianOutputTokens:        u.MedianOutputTokens,
		MedianCacheCreationTokens: u.MedianCacheCreationTokens,
		MedianCacheReadTokens:     u.MedianCacheReadTokens,
	}
}

func buildJSONRankings(sr *result.SuiteResult, weights Weights) []jsonRanking {
	showQuality := hasComparison(sr)
	ranks := RankVariants(sr, weights)
	var rankings []jsonRanking
	for _, rank := range ranks {
		jr := jsonRanking{
			Rank:           rank.Rank,
			Name:           rank.Name,
			Runner:         rank.Runner,
			Model:          rank.Model,
			CompositeScore: rank.CompositeScore,
			PassRate:       rank.PassRate,
			MedianCostUSD:  rank.MedianCostUSD,
			MedianDuration: rank.MedianDuration,
		}
		if showQuality {
			q := rank.QualityScore
			jr.QualityScore = &q
		}
		rankings = append(rankings, jr)
	}
	return rankings
}
