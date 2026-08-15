package report

import "github.com/driangle/skival/internal/result"

type jsonRanking struct {
	Rank              int      `json:"rank"`
	Name              string   `json:"name"`
	Runner            string   `json:"runner,omitempty"`
	Model             string   `json:"model,omitempty"`
	CompositeScore    float64  `json:"composite_score"`
	PassRate          float64  `json:"pass_rate"`
	PassRateLow       float64  `json:"pass_rate_low"`
	PassRateHigh      float64  `json:"pass_rate_high"`
	MedianCostUSD     float64  `json:"median_cost_usd"`
	MedianDuration    int64    `json:"median_duration_ms"`
	MedianTotalTokens *int64   `json:"median_total_tokens,omitempty"`
	QualityScore      *float64 `json:"quality_score,omitempty"`
}

func buildJSONRankings(sr *result.SuiteResult, weights Weights) []jsonRanking {
	showQuality := hasComparison(sr)
	showTokens := weights.Tokens > 0
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
			PassRateLow:    rank.PassLow,
			PassRateHigh:   rank.PassHigh,
			MedianCostUSD:  rank.MedianCostUSD,
			MedianDuration: rank.MedianDuration,
		}
		if showTokens {
			t := rank.MedianTotalTokens
			jr.MedianTotalTokens = &t
		}
		if showQuality {
			q := rank.QualityScore
			jr.QualityScore = &q
		}
		rankings = append(rankings, jr)
	}
	return rankings
}
