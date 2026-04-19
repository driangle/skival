package compare

import (
	"github.com/driangle/skival/internal/result"
)

// Comparison holds the result of comparing a baseline and candidate suite run.
type Comparison struct {
	Baseline  RunMeta           `json:"baseline"`
	Candidate RunMeta           `json:"candidate"`
	Evals     []EvalComparison  `json:"evals"`
}

// RunMeta captures identifying info about one side of the comparison.
type RunMeta struct {
	Description string `json:"description"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
}

// EvalComparison holds per-variant deltas for a single eval.
type EvalComparison struct {
	EvalID   string              `json:"eval_id"`
	EvalName string              `json:"eval_name"`
	Variants []VariantComparison `json:"variants"`
}

// VariantComparison holds the delta between baseline and candidate for one variant.
type VariantComparison struct {
	Name   string          `json:"name"`
	Status ComparisonStatus `json:"status"` // "matched", "added", "removed"

	// Present only when Status == "matched".
	PassRateDelta    *float64 `json:"pass_rate_delta_pp,omitempty"`    // percentage points
	CostDelta        *float64 `json:"cost_delta_usd,omitempty"`       // absolute USD change
	CostDeltaPct     *float64 `json:"cost_delta_pct,omitempty"`       // percent change
	DurationDeltaMs  *int64   `json:"duration_delta_ms,omitempty"`    // absolute ms change
	DurationDeltaPct *float64 `json:"duration_delta_pct,omitempty"`   // percent change

	BaselinePassRate  *float64 `json:"baseline_pass_rate,omitempty"`
	CandidatePassRate *float64 `json:"candidate_pass_rate,omitempty"`
	BaselineCost      *float64 `json:"baseline_median_cost,omitempty"`
	CandidateCost     *float64 `json:"candidate_median_cost,omitempty"`
	BaselineDuration  *int64   `json:"baseline_median_duration_ms,omitempty"`
	CandidateDuration *int64   `json:"candidate_median_duration_ms,omitempty"`
}

// ComparisonStatus indicates whether a variant was matched, added, or removed.
type ComparisonStatus string

const (
	StatusMatched ComparisonStatus = "matched"
	StatusAdded   ComparisonStatus = "added"
	StatusRemoved ComparisonStatus = "removed"
)

// Compare produces a diff between baseline and candidate suite results.
func Compare(baseline, candidate *result.SuiteResult) *Comparison {
	c := &Comparison{
		Baseline: RunMeta{
			Description: baseline.Description,
			StartedAt:   baseline.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			FinishedAt:  baseline.FinishedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Candidate: RunMeta{
			Description: candidate.Description,
			StartedAt:   candidate.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			FinishedAt:  candidate.FinishedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	// Index candidate evals by ID for lookup.
	candEvals := make(map[string]*result.EvalResult)
	for i := range candidate.Evals {
		candEvals[candidate.Evals[i].EvalID] = &candidate.Evals[i]
	}

	// Track which candidate evals we've seen.
	seen := make(map[string]bool)

	// Process baseline evals.
	for _, bEval := range baseline.Evals {
		seen[bEval.EvalID] = true
		cEval, ok := candEvals[bEval.EvalID]
		if !ok {
			// Eval exists in baseline only — all variants are "removed".
			ec := EvalComparison{EvalID: bEval.EvalID, EvalName: bEval.EvalName}
			for _, bt := range bEval.Variants {
				ec.Variants = append(ec.Variants, VariantComparison{
					Name:   bt.Name,
					Status: StatusRemoved,
				})
			}
			c.Evals = append(c.Evals, ec)
			continue
		}

		c.Evals = append(c.Evals, compareEval(bEval, *cEval))
	}

	// Process candidate-only evals.
	for _, cEval := range candidate.Evals {
		if seen[cEval.EvalID] {
			continue
		}
		ec := EvalComparison{EvalID: cEval.EvalID, EvalName: cEval.EvalName}
		for _, ct := range cEval.Variants {
			ec.Variants = append(ec.Variants, VariantComparison{
				Name:   ct.Name,
				Status: StatusAdded,
			})
		}
		c.Evals = append(c.Evals, ec)
	}

	return c
}

func compareEval(baseline, candidate result.EvalResult) EvalComparison {
	ec := EvalComparison{EvalID: baseline.EvalID, EvalName: baseline.EvalName}

	candVars := make(map[string]*result.VariantResult)
	for i := range candidate.Variants {
		candVars[candidate.Variants[i].Name] = &candidate.Variants[i]
	}

	seen := make(map[string]bool)

	for _, bt := range baseline.Variants {
		seen[bt.Name] = true
		ct, ok := candVars[bt.Name]
		if !ok {
			ec.Variants = append(ec.Variants, VariantComparison{
				Name:   bt.Name,
				Status: StatusRemoved,
			})
			continue
		}
		ec.Variants = append(ec.Variants, compareVariant(bt, *ct))
	}

	for _, ct := range candidate.Variants {
		if seen[ct.Name] {
			continue
		}
		ec.Variants = append(ec.Variants, VariantComparison{
			Name:   ct.Name,
			Status: StatusAdded,
		})
	}

	return ec
}

func compareVariant(baseline, candidate result.VariantResult) VariantComparison {
	tc := VariantComparison{
		Name:   baseline.Name,
		Status: StatusMatched,
	}

	bPass := passRate(baseline.Runs)
	cPass := passRate(candidate.Runs)
	if bPass != nil && cPass != nil {
		delta := *cPass - *bPass
		tc.PassRateDelta = &delta
	}
	tc.BaselinePassRate = bPass
	tc.CandidatePassRate = cPass

	bCost := medianCost(baseline)
	cCost := medianCost(candidate)
	if bCost != nil && cCost != nil {
		delta := *cCost - *bCost
		tc.CostDelta = &delta
		if *bCost != 0 {
			pct := delta / *bCost * 100
			tc.CostDeltaPct = &pct
		}
	}
	tc.BaselineCost = bCost
	tc.CandidateCost = cCost

	bDur := medianDuration(baseline)
	cDur := medianDuration(candidate)
	if bDur != nil && cDur != nil {
		delta := *cDur - *bDur
		tc.DurationDeltaMs = &delta
		if *bDur != 0 {
			pct := float64(delta) / float64(*bDur) * 100
			tc.DurationDeltaPct = &pct
		}
	}
	tc.BaselineDuration = bDur
	tc.CandidateDuration = cDur

	return tc
}

// passRate computes the pass rate from runs. Returns nil if no runs have Pass set.
func passRate(runs []result.RunResult) *float64 {
	var passed, verified int
	for _, r := range runs {
		if r.Pass != nil {
			verified++
			if *r.Pass {
				passed++
			}
		}
	}
	if verified == 0 {
		return nil
	}
	rate := float64(passed) / float64(verified)
	return &rate
}

func medianCost(t result.VariantResult) *float64 {
	if t.Aggregate != nil {
		return &t.Aggregate.MedianCostUSD
	}
	if len(t.Runs) == 0 {
		return nil
	}
	agg := result.ComputeAggregate(t.Runs)
	if agg == nil {
		return nil
	}
	return &agg.MedianCostUSD
}

func medianDuration(t result.VariantResult) *int64 {
	if t.Aggregate != nil {
		return &t.Aggregate.MedianDurationMs
	}
	if len(t.Runs) == 0 {
		return nil
	}
	agg := result.ComputeAggregate(t.Runs)
	if agg == nil {
		return nil
	}
	return &agg.MedianDurationMs
}
