package result

import (
	"math"
	"sort"
)

// Aggregate holds computed statistics across multiple sample runs.
type Aggregate struct {
	MedianCostUSD    float64
	MinCostUSD       float64
	MaxCostUSD       float64
	MedianDurationMs int64
	MinDurationMs    int64
	MaxDurationMs    int64
	CostCV           *float64
	DurationCV       *float64
	Pass             *bool
}

// ComputeAggregate calculates aggregate statistics from a set of runs.
// Returns nil if runs is empty.
func ComputeAggregate(runs []RunResult) *Aggregate {
	if len(runs) == 0 {
		return nil
	}

	costs := make([]float64, len(runs))
	durations := make([]float64, len(runs))
	for i, r := range runs {
		costs[i] = r.CostUSD
		durations[i] = float64(r.DurationMs)
	}

	agg := &Aggregate{
		MedianCostUSD:    median(costs),
		MinCostUSD:       minVal(costs),
		MaxCostUSD:       maxVal(costs),
		MedianDurationMs: int64(median(durations)),
		MinDurationMs:    int64(minVal(durations)),
		MaxDurationMs:    int64(maxVal(durations)),
		CostCV:           cv(costs),
		DurationCV:       cv(durations),
		Pass:             conservativePass(runs),
	}

	return agg
}

func median(vals []float64) float64 {
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func minVal(vals []float64) float64 {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxVal(vals []float64) float64 {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// cv returns the coefficient of variation (population stddev / mean).
// Returns nil if fewer than 3 values or mean is zero.
func cv(vals []float64) *float64 {
	if len(vals) < 3 {
		return nil
	}

	var sum float64
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))

	if mean == 0 {
		return nil
	}

	var sqDiffSum float64
	for _, v := range vals {
		d := v - mean
		sqDiffSum += d * d
	}
	stddev := math.Sqrt(sqDiffSum / float64(len(vals)))

	result := stddev / mean
	return &result
}

// conservativePass returns the most conservative pass result across runs.
// Priority: fail > nil (unverified) > pass.
// Returns nil if no runs have a Pass value set.
func conservativePass(runs []RunResult) *bool {
	hasAny := false
	allPass := true

	for _, r := range runs {
		if r.Pass == nil {
			allPass = false
			continue
		}
		hasAny = true
		if !*r.Pass {
			f := false
			return &f
		}
	}

	if !hasAny {
		return nil
	}

	if !allPass {
		// Some nil (unverified) mixed with passes — conservative: nil
		return nil
	}

	t := true
	return &t
}
