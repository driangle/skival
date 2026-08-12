package compare

import (
	"github.com/driangle/skival/internal/result"
)

// Comparison holds the result of comparing a baseline and candidate suite run.
type Comparison struct {
	Baseline  RunMeta          `json:"baseline"`
	Candidate RunMeta          `json:"candidate"`
	Evals     []EvalComparison `json:"evals"`
}

// RunMeta captures identifying info about one side of the comparison.
type RunMeta struct {
	Title       string `json:"title,omitempty"`
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
	Name   string           `json:"name"`
	Status ComparisonStatus `json:"status"` // "matched", "added", "removed"

	// Present only when Status == "matched".
	PassRateDelta    *float64 `json:"pass_rate_delta_pp,omitempty"` // percentage points
	CostDelta        *float64 `json:"cost_delta_usd,omitempty"`     // absolute USD change
	CostDeltaPct     *float64 `json:"cost_delta_pct,omitempty"`     // percent change
	DurationDeltaMs  *int64   `json:"duration_delta_ms,omitempty"`  // absolute ms change
	DurationDeltaPct *float64 `json:"duration_delta_pct,omitempty"` // percent change

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
		Baseline:  newRunMeta(baseline),
		Candidate: newRunMeta(candidate),
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
			c.Evals = append(c.Evals, evalWithStatus(bEval, StatusRemoved))
			continue
		}
		c.Evals = append(c.Evals, compareEval(bEval, *cEval))
	}

	// Process candidate-only evals — all variants are "added".
	for _, cEval := range candidate.Evals {
		if seen[cEval.EvalID] {
			continue
		}
		c.Evals = append(c.Evals, evalWithStatus(cEval, StatusAdded))
	}

	return c
}

// newRunMeta builds the identifying metadata for one side of the comparison.
func newRunMeta(sr *result.SuiteResult) RunMeta {
	const layout = "2006-01-02T15:04:05Z07:00"
	return RunMeta{
		Title:       sr.Title,
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format(layout),
		FinishedAt:  sr.FinishedAt.Format(layout),
	}
}

// evalWithStatus builds an EvalComparison whose variants all share the given
// status, used for evals present on only one side of the comparison.
func evalWithStatus(eval result.EvalResult, status ComparisonStatus) EvalComparison {
	ec := EvalComparison{EvalID: eval.EvalID, EvalName: eval.EvalName}
	for _, v := range eval.Variants {
		ec.Variants = append(ec.Variants, VariantComparison{
			Name:   v.Name,
			Status: status,
		})
	}
	return ec
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
	setPassRateDeltas(&tc, baseline, candidate)
	setCostDeltas(&tc, baseline, candidate)
	setDurationDeltas(&tc, baseline, candidate)
	return tc
}

// setPassRateDeltas fills in the pass-rate fields on tc.
func setPassRateDeltas(tc *VariantComparison, baseline, candidate result.VariantResult) {
	bPass := passRate(baseline.Runs)
	cPass := passRate(candidate.Runs)
	if bPass != nil && cPass != nil {
		delta := *cPass - *bPass
		tc.PassRateDelta = &delta
	}
	tc.BaselinePassRate = bPass
	tc.CandidatePassRate = cPass
}

// setCostDeltas fills in the median-cost fields on tc.
func setCostDeltas(tc *VariantComparison, baseline, candidate result.VariantResult) {
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
}

// setDurationDeltas fills in the median-duration fields on tc.
func setDurationDeltas(tc *VariantComparison, baseline, candidate result.VariantResult) {
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
