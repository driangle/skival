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
	// Usage holds median token usage across the runs, or nil when no run
	// reported any tokens (e.g. the exec runner), so token-free suites render
	// exactly as before.
	Usage *UsageAggregate
	Pass  *bool
}

// UsageAggregate holds median token usage across a variant's runs.
type UsageAggregate struct {
	MedianInputTokens         int64
	MedianOutputTokens        int64
	MedianCacheCreationTokens int64
	MedianCacheReadTokens     int64
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
		Usage:            computeUsageAggregate(runs),
		Pass:             conservativePass(runs),
	}

	return agg
}

// computeUsageAggregate returns median token usage across runs, or nil when no
// run reported any tokens.
func computeUsageAggregate(runs []RunResult) *UsageAggregate {
	in := make([]float64, len(runs))
	out := make([]float64, len(runs))
	cc := make([]float64, len(runs))
	cr := make([]float64, len(runs))
	var any bool
	for i, r := range runs {
		u := r.Usage
		in[i] = float64(u.InputTokens)
		out[i] = float64(u.OutputTokens)
		cc[i] = float64(u.CacheCreationInputTokens)
		cr[i] = float64(u.CacheReadInputTokens)
		if u.InputTokens != 0 || u.OutputTokens != 0 ||
			u.CacheCreationInputTokens != 0 || u.CacheReadInputTokens != 0 {
			any = true
		}
	}
	if !any {
		return nil
	}
	return &UsageAggregate{
		MedianInputTokens:         int64(median(in)),
		MedianOutputTokens:        int64(median(out)),
		MedianCacheCreationTokens: int64(median(cc)),
		MedianCacheReadTokens:     int64(median(cr)),
	}
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
//
// Returns nil for fewer than 3 values: CV is a measure of run-to-run
// variability, and with only 1–2 samples that estimate is uninformative
// (n=1 has no spread at all, n=2 is a single pair whose dispersion is too
// unstable to report as a variability signal). A nil CV therefore means
// "not enough samples to say", not "zero variance". It also returns nil
// when the mean is zero, since CV normalizes by the mean.
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
