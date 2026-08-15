package executor

import (
	"context"
	"sync"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// runSamples runs a variant's samples sequentially or with bounded concurrency,
// returning results in sample order. Once the budget cap is crossed, no further
// samples are started, so fewer than `samples` results may be returned.
func runSamples(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, samples, parallel int, runner agentrunner.Runner, prog *progress, timeoutOverride int, b *budget) []result.RunResult {
	if parallel <= 1 {
		return runSamplesSequential(ctx, eval, label, v, samples, runner, prog, timeoutOverride, b)
	}
	return runSamplesParallel(ctx, eval, label, v, samples, parallel, runner, prog, timeoutOverride, b)
}

// runSamplesSequential runs samples one at a time, stopping before the next
// sample once the budget cap is crossed.
func runSamplesSequential(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, samples int, runner agentrunner.Runner, prog *progress, timeoutOverride int, b *budget) []result.RunResult {
	var runs []result.RunResult
	for i := 0; i < samples; i++ {
		if b.stopped() {
			break
		}
		run := runSample(ctx, eval, label, v, i, samples, runner, prog, timeoutOverride)
		b.add(run.CostUSD)
		runs = append(runs, run)
		if i == 0 {
			preflightToolCheck(v, run, prog)
		}
	}
	return runs
}

// runSamplesParallel runs samples with bounded concurrency. Each worker checks
// the budget after acquiring its slot and skips the sample if the cap is
// already crossed; skipped slots are dropped from the returned slice.
func runSamplesParallel(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, samples, parallel int, runner agentrunner.Runner, prog *progress, timeoutOverride int, b *budget) []result.RunResult {
	runs := make([]*result.RunResult, samples)
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	for i := 0; i < samples; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if b.stopped() {
				return
			}
			run := runSample(ctx, eval, label, v, idx, samples, runner, prog, timeoutOverride)
			b.add(run.CostUSD)
			runs[idx] = &run
		}(i)
	}
	wg.Wait()
	out := compactRuns(runs)
	if len(out) > 0 {
		preflightToolCheck(v, out[0], prog)
	}
	return out
}

// compactRuns drops nil (un-started) entries, preserving sample order.
func compactRuns(runs []*result.RunResult) []result.RunResult {
	var out []result.RunResult
	for _, r := range runs {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}
