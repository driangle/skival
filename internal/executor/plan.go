package executor

import (
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// PlanEntry is one cell of the resolved run matrix: a single variant of a
// single eval, with the model/runner/sample count that would be used.
type PlanEntry struct {
	EvalID   string
	EvalName string
	Variant  string
	Runner   string
	Model    string
	Samples  int
	// EstCostUSD is the estimated total cost for this cell (prior median cost ×
	// samples). Nil when no prior median was available for the eval+variant.
	EstCostUSD *float64
}

// Plan is the resolved run matrix for a suite under a given set of options. It
// is what `--dry-run` prints: evals × variants × samples, without executing.
type Plan struct {
	Entries      []PlanEntry
	TotalSamples int
	// EstTotalUSD is the summed estimate across cells that had a prior median.
	// Nil until ApplyEstimate finds at least one matching prior.
	EstTotalUSD *float64
	// EstimatedCells / TotalCells report estimate coverage so the renderer can
	// flag when only some cells could be priced.
	EstimatedCells int
	TotalCells     int
}

// BuildPlan resolves the run matrix for the suite under opts, applying the same
// eval/variant filters and sample resolution the executor uses — but running
// nothing. Filter typos are rejected up front, exactly as Execute does.
func BuildPlan(s *suite.Suite, opts *Options) (*Plan, error) {
	if opts == nil {
		opts = &Options{}
	}
	if err := validateFilters(s, opts); err != nil {
		return nil, err
	}

	plan := &Plan{}
	evals := filterEvals(s.Evals, opts.EvalIDs)
	for i := range evals {
		eval := &evals[i]
		samples := resolveSamples(eval, opts)
		for _, ve := range collectVariants(eval, opts.Variants) {
			plan.Entries = append(plan.Entries, PlanEntry{
				EvalID:   eval.ID,
				EvalName: evalLabel(eval),
				Variant:  ve.variant.Name,
				Runner:   ve.variant.Runner,
				Model:    ve.variant.Model,
				Samples:  samples,
			})
			plan.TotalSamples += samples
		}
	}
	plan.TotalCells = len(plan.Entries)
	return plan, nil
}

// CostPriors maps eval ID -> variant name -> prior median cost (USD/sample).
type CostPriors map[string]map[string]float64

// PriorsFromResults extracts per-eval/variant median sample cost from a
// previously persisted (or in-memory) suite result, for use as a dry-run
// estimate source.
func PriorsFromResults(sr *result.SuiteResult) CostPriors {
	priors := make(CostPriors)
	if sr == nil {
		return priors
	}
	for _, er := range sr.Evals {
		for _, vr := range er.Variants {
			if vr.Aggregate == nil {
				continue
			}
			if _, ok := priors[er.EvalID]; !ok {
				priors[er.EvalID] = make(map[string]float64)
			}
			priors[er.EvalID][vr.Name] = vr.Aggregate.MedianCostUSD
		}
	}
	return priors
}

// ApplyEstimate prices each cell using the matching prior median × its sample
// count, and sums the priced cells into EstTotalUSD. Cells with no prior are
// left unpriced (EstCostUSD nil) and excluded from the total.
func (p *Plan) ApplyEstimate(priors CostPriors) {
	var total float64
	var priced int
	for i := range p.Entries {
		e := &p.Entries[i]
		median, ok := lookupPrior(priors, e.EvalID, e.Variant)
		if !ok {
			continue
		}
		cost := median * float64(e.Samples)
		e.EstCostUSD = &cost
		total += cost
		priced++
	}
	p.EstimatedCells = priced
	if priced > 0 {
		p.EstTotalUSD = &total
	}
}

func lookupPrior(priors CostPriors, evalID, variant string) (float64, bool) {
	variants, ok := priors[evalID]
	if !ok {
		return 0, false
	}
	median, ok := variants[variant]
	return median, ok
}
