package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// Execute runs all evals in the suite, returning collected results.
// Runner errors are captured per-run and do not abort the suite.
func Execute(ctx context.Context, s *suite.Suite, reg *registry.Registry, opts *Options) (*result.SuiteResult, error) {
	if opts == nil {
		opts = &Options{}
	}

	if err := validateFilters(s, opts); err != nil {
		return nil, err
	}

	prog := newProgress(opts.Progress)

	sr := &result.SuiteResult{
		Title:       s.Title,
		Description: s.Description,
		StartedAt:   time.Now(),
	}

	evals := filterEvals(s.Evals, opts.EvalIDs)
	b := newBudget(opts.MaxCost)

	for i := range evals {
		// Once the cost cap is crossed, stop before starting a new eval so we
		// don't run its (side-effecting) before hook only to skip every sample.
		if b.stopped() {
			break
		}
		prog.evalStart(i+1, len(evals), evalLabel(&evals[i]))
		evalResult := executeEval(ctx, &evals[i], s.Compare, reg, opts, prog, b)
		sr.Evals = append(sr.Evals, evalResult)
	}

	sr.FinishedAt = time.Now()
	sr.Abort = budgetAbort(b)
	prog.finish()
	cleanupWorkdirs(sr, opts.KeepWorkdirs)
	return sr, nil
}

// budgetAbort returns an Abort record when the cost cap was crossed, else nil.
func budgetAbort(b *budget) *result.Abort {
	spent, exceeded := b.report()
	if !exceeded {
		return nil
	}
	return &result.Abort{Reason: "cost cap exceeded", SpentUSD: spent, CapUSD: b.cap}
}

func executeEval(ctx context.Context, eval *suite.Eval, suiteCmp *suite.Compare, reg *registry.Registry, opts *Options, prog *progress, b *budget) result.EvalResult {
	er := result.EvalResult{
		EvalID:   eval.ID,
		EvalName: eval.Name,
	}

	label := evalLabel(eval)

	// Always run after hook, even on error.
	defer runAfterHook(ctx, eval.Setup, eval.Dir)

	variants := collectVariants(eval, opts.Variants)

	// Run before hook once before any variant.
	if err := runBeforeHook(ctx, eval.Setup, eval.Dir); err != nil {
		return evalBeforeHookFailed(&er, variants, label, err, prog)
	}

	pv := opts.ParallelVariants
	if pv > len(variants) {
		pv = len(variants)
	}

	if pv > 1 {
		er.Variants = append(er.Variants, runVariantsParallel(ctx, eval, label, variants, reg, opts, prog, pv, b)...)
	} else if runVariantsSequential(ctx, eval, label, variants, reg, opts, prog, &er, b) {
		return er
	}

	// Comparative judging runs once, after every variant of the eval has been
	// verified, scoring the passing variants against each other.
	runComparison(ctx, eval, suiteCmp, &er, reg, opts.Compare)

	return er
}

// evalBeforeHookFailed records that the before hook failed, marking every
// variant as skipped and reporting progress.
func evalBeforeHookFailed(er *result.EvalResult, variants []variantEntry, label string, err error, prog *progress) result.EvalResult {
	er.Err = err
	for _, v := range variants {
		er.Skipped = append(er.Skipped, result.SkippedVariant{
			Name:   v.variant.Name,
			Reason: "before hook failed",
		})
	}
	prog.evalError(label, err)
	prog.skippedVariants(label, er.Skipped)
	return *er
}

// runVariantsParallel executes variants concurrently with bounded concurrency,
// returning results in variant order.
func runVariantsParallel(ctx context.Context, eval *suite.Eval, label string, variants []variantEntry, reg *registry.Registry, opts *Options, prog *progress, limit int, b *budget) []result.VariantResult {
	results := make([]result.VariantResult, len(variants))
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup

	for i := range variants {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cache := make(map[string]agentrunner.Runner)
			v := variants[idx]
			results[idx] = executeVariant(ctx, eval, label, v.variant, v.isControl, reg, cache, opts, prog, b)
		}(i)
	}
	wg.Wait()
	return results
}

// runVariantsSequential executes variants one at a time, running the reset hook
// between variants. It appends results to er and returns true if the eval was
// aborted early (reset hook failure), in which case remaining variants are skipped.
func runVariantsSequential(ctx context.Context, eval *suite.Eval, label string, variants []variantEntry, reg *registry.Registry, opts *Options, prog *progress, er *result.EvalResult, b *budget) bool {
	runnerCache := make(map[string]agentrunner.Runner)

	for i := range variants {
		// Run reset between variants (not before the first one).
		if i > 0 {
			if err := runResetHook(ctx, eval.Setup, eval.Dir); err != nil {
				er.Err = fmt.Errorf("reset hook failed between variant %q and %q: %w",
					variants[i-1].variant.Name, variants[i].variant.Name, err)
				for _, v := range variants[i:] {
					er.Skipped = append(er.Skipped, result.SkippedVariant{
						Name:   v.variant.Name,
						Reason: fmt.Sprintf("reset hook failed after variant %q", variants[i-1].variant.Name),
					})
				}
				prog.evalError(label, er.Err)
				prog.skippedVariants(label, er.Skipped)
				return true
			}
		}

		v := variants[i]
		vr := executeVariant(ctx, eval, label, v.variant, v.isControl, reg, runnerCache, opts, prog, b)
		er.Variants = append(er.Variants, vr)
	}
	return false
}

type variantEntry struct {
	variant   *suite.Variant
	isControl bool
}

func collectVariants(eval *suite.Eval, filter []string) []variantEntry {
	filterSet := toSet(filter)

	var entries []variantEntry

	for i := range eval.Variants {
		if shouldInclude(eval.Variants[i].Name, filterSet) {
			entries = append(entries, variantEntry{&eval.Variants[i], i == 0})
		}
	}

	return entries
}

func executeVariant(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, isControl bool, reg *registry.Registry, runnerCache map[string]agentrunner.Runner, opts *Options, prog *progress, b *budget) result.VariantResult {
	vr := result.VariantResult{
		Name:      v.Name,
		Runner:    v.Runner,
		Model:     v.Model,
		IsControl: isControl,
	}

	runner, ok := resolveVariantRunner(runnerCache, v, reg, &vr)
	if !ok {
		return vr
	}

	samples := resolveSamples(eval, opts)
	parallel := resolveParallel(eval, opts, samples)

	vr.Runs = runSamples(ctx, eval, label, v, samples, parallel, runner, prog, opts.Timeout, b)
	vr.Aggregate = result.ComputeAggregate(vr.Runs)

	return vr
}

// resolveVariantRunner returns the runner for a variant, creating and caching it
// on first use. On creation failure it records the error on vr and returns false.
func resolveVariantRunner(cache map[string]agentrunner.Runner, v *suite.Variant, reg *registry.Registry, vr *result.VariantResult) (agentrunner.Runner, bool) {
	if runner, ok := cache[v.Runner]; ok {
		return runner, true
	}
	runner, err := reg.Create(v.Runner, v.RunnerConfig)
	if err != nil {
		slog.Error("Failed to create runner", "runner", v.Runner, "variant", v.Name, "err", err)
		vr.Runs = append(vr.Runs, result.RunResult{Sample: 1, Err: fmt.Errorf("creating runner %q: %w", v.Runner, err)})
		return nil, false
	}
	cache[v.Runner] = runner
	return runner, true
}

// resolveSamples resolves the sample count: CLI override > eval-level > default of 1.
func resolveSamples(eval *suite.Eval, opts *Options) int {
	samples := 1
	if eval.Samples != nil {
		samples = *eval.Samples
	}
	if opts.Samples > 0 {
		samples = opts.Samples
	}
	return samples
}

// resolveParallel resolves the parallelism: CLI override > eval-level > default of 1,
// clamped to the sample count.
func resolveParallel(eval *suite.Eval, opts *Options, samples int) int {
	parallel := 1
	if eval.Parallel != nil {
		parallel = *eval.Parallel
	}
	if opts.Parallel > 0 {
		parallel = opts.Parallel
	}
	if parallel > samples {
		parallel = samples
	}
	return parallel
}
