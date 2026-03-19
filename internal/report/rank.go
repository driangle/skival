package report

import (
	"sort"

	"github.com/driangle/skival/internal/result"
)

const (
	WeightPass     = 0.60
	WeightCost     = 0.28
	WeightDuration = 0.12
)

// TreatmentRank holds the ranking data for a single treatment.
type TreatmentRank struct {
	Name           string
	PassRate       float64
	MedianCostUSD  float64
	MedianDuration int64
	CompositeScore float64
	Rank           int
}

// RankTreatments computes a weighted composite score for each treatment
// across all evals in a suite result and returns them sorted best-first.
func RankTreatments(sr *result.SuiteResult) []TreatmentRank {
	stats := collectStats(sr)
	if len(stats) == 0 {
		return nil
	}

	ranks := make([]TreatmentRank, 0, len(stats))
	for name, s := range stats {
		ranks = append(ranks, TreatmentRank{
			Name:           name,
			PassRate:       s.passRate(),
			MedianCostUSD:  s.medianCost(),
			MedianDuration: s.medianDuration(),
		})
	}

	normalize(ranks)

	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].CompositeScore != ranks[j].CompositeScore {
			return ranks[i].CompositeScore > ranks[j].CompositeScore
		}
		return ranks[i].Name < ranks[j].Name
	})

	for i := range ranks {
		ranks[i].Rank = i + 1
	}

	return ranks
}

// treatmentStats accumulates raw data for a treatment across evals.
type treatmentStats struct {
	passed    int
	verified  int
	costs     []float64
	durations []int64
}

func (s *treatmentStats) passRate() float64 {
	if s.verified == 0 {
		return 0
	}
	return float64(s.passed) / float64(s.verified)
}

func (s *treatmentStats) medianCost() float64 {
	if len(s.costs) == 0 {
		return 0
	}
	sorted := make([]float64, len(s.costs))
	copy(sorted, s.costs)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func (s *treatmentStats) medianDuration() int64 {
	if len(s.durations) == 0 {
		return 0
	}
	sorted := make([]int64, len(s.durations))
	copy(sorted, s.durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func collectStats(sr *result.SuiteResult) map[string]*treatmentStats {
	stats := make(map[string]*treatmentStats)

	for _, eval := range sr.Evals {
		for _, treat := range eval.Treatments {
			s, ok := stats[treat.Name]
			if !ok {
				s = &treatmentStats{}
				stats[treat.Name] = s
			}

			for _, run := range treat.Runs {
				s.costs = append(s.costs, run.CostUSD)
				s.durations = append(s.durations, run.DurationMs)

				if run.Pass != nil {
					s.verified++
					if *run.Pass {
						s.passed++
					}
				}
			}
		}
	}

	return stats
}

// normalize computes composite scores. For pass rate, higher is better (1.0 = best).
// For cost and duration, lower is better, so normalization is inverted.
func normalize(ranks []TreatmentRank) {
	if len(ranks) == 0 {
		return
	}

	var minCost, maxCost float64
	var minDur, maxDur int64
	var minPass, maxPass float64

	for i, r := range ranks {
		if i == 0 {
			minCost, maxCost = r.MedianCostUSD, r.MedianCostUSD
			minDur, maxDur = r.MedianDuration, r.MedianDuration
			minPass, maxPass = r.PassRate, r.PassRate
			continue
		}
		if r.MedianCostUSD < minCost {
			minCost = r.MedianCostUSD
		}
		if r.MedianCostUSD > maxCost {
			maxCost = r.MedianCostUSD
		}
		if r.MedianDuration < minDur {
			minDur = r.MedianDuration
		}
		if r.MedianDuration > maxDur {
			maxDur = r.MedianDuration
		}
		if r.PassRate < minPass {
			minPass = r.PassRate
		}
		if r.PassRate > maxPass {
			maxPass = r.PassRate
		}
	}

	for i := range ranks {
		passNorm := normHigherBetter(ranks[i].PassRate, minPass, maxPass)
		costNorm := normLowerBetter(ranks[i].MedianCostUSD, minCost, maxCost)
		durNorm := normLowerBetter(float64(ranks[i].MedianDuration), float64(minDur), float64(maxDur))

		ranks[i].CompositeScore = WeightPass*passNorm + WeightCost*costNorm + WeightDuration*durNorm
	}
}

// normHigherBetter returns 1.0 for the max value, 0.0 for the min.
// If all values are equal, returns 1.0.
func normHigherBetter(val, min, max float64) float64 {
	if max == min {
		return 1.0
	}
	return (val - min) / (max - min)
}

// normLowerBetter returns 1.0 for the min value, 0.0 for the max.
// If all values are equal, returns 1.0.
func normLowerBetter(val, min, max float64) float64 {
	if max == min {
		return 1.0
	}
	return 1.0 - (val-min)/(max-min)
}
