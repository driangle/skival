package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	agentrunner "github.com/driangle/agent-runner/go"
	"github.com/driangle/agent-runner/go/claudecode"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/verifier"
)

// Execute runs all evals in the suite, returning collected results.
// Runner errors are captured per-run and do not abort the suite.
func Execute(ctx context.Context, s *suite.Suite, runner agentrunner.Runner, opts *Options) (*result.SuiteResult, error) {
	if opts == nil {
		opts = &Options{}
	}

	prog := newProgress(opts.Progress)

	sr := &result.SuiteResult{
		Description: s.Description,
		StartedAt:   time.Now(),
	}

	evals := filterEvals(s.Evals, opts.EvalIDs)

	for i := range evals {
		prog.evalStart(i+1, len(evals), evals[i].Name)
		evalResult := executeEval(ctx, &evals[i], runner, opts, prog)
		sr.Evals = append(sr.Evals, evalResult)
	}

	sr.FinishedAt = time.Now()
	prog.finish()
	return sr, nil
}

func executeEval(ctx context.Context, eval *suite.Eval, runner agentrunner.Runner, opts *Options, prog *progress) result.EvalResult {
	er := result.EvalResult{
		EvalID:   eval.ID,
		EvalName: eval.Name,
	}

	// Always run after hook, even on error.
	defer runAfterHook(ctx, eval.Setup, eval.Dir)

	// Run before hook once before any treatment.
	if err := runBeforeHook(ctx, eval.Setup, eval.Dir); err != nil {
		er.Err = err
		return er
	}

	treatments := collectTreatments(eval, opts.Treatments)

	for i := range treatments {
		// Run reset between treatments (not before the first one).
		if i > 0 {
			if err := runResetHook(ctx, eval.Setup, eval.Dir); err != nil {
				er.Err = err
				return er
			}
		}

		t := treatments[i]
		tr := executeTreatment(ctx, eval, t.treatment, t.isControl, runner, prog)
		er.Treatments = append(er.Treatments, tr)
	}

	return er
}

type treatmentEntry struct {
	treatment *suite.Treatment
	isControl bool
}

func collectTreatments(eval *suite.Eval, filter []string) []treatmentEntry {
	filterSet := toSet(filter)

	var entries []treatmentEntry

	if shouldInclude(eval.Treatments.Control.Name, filterSet) {
		entries = append(entries, treatmentEntry{&eval.Treatments.Control, true})
	}
	for i := range eval.Treatments.Variations {
		if shouldInclude(eval.Treatments.Variations[i].Name, filterSet) {
			entries = append(entries, treatmentEntry{&eval.Treatments.Variations[i], false})
		}
	}

	return entries
}

func executeTreatment(ctx context.Context, eval *suite.Eval, t *suite.Treatment, isControl bool, runner agentrunner.Runner, prog *progress) result.TreatmentResult {
	tr := result.TreatmentResult{
		Name:      t.Name,
		IsControl: isControl,
	}

	samples := 1
	if eval.Samples != nil {
		samples = *eval.Samples
	}

	var pipelineOpts []verifier.PipelineOption
	if len(eval.Correctness.Judge) > 0 {
		pipelineOpts = append(pipelineOpts, verifier.WithJudge(runner, eval.Prompt))
	}
	pipeline := verifier.BuildPipeline(eval.Correctness, eval.Dir, pipelineOpts...)

	for i := 0; i < samples; i++ {
		prog.sampleStart(eval.Name, t.Name, i+1, samples)
		slog.Debug("Running sample", "eval", eval.Name, "treatment", t.Name, "sample", i+1, "total", samples)
		run := executeSingleRun(ctx, eval, t, i+1, runner)
		if run.Err != nil {
			slog.Debug("Sample error", "eval", eval.Name, "treatment", t.Name, "sample", i+1, "err", run.Err)
		} else {
			slog.Debug("Sample complete", "eval", eval.Name, "treatment", t.Name, "sample", i+1,
				"cost", run.CostUSD, "duration_ms", run.DurationMs, "exit_code", run.ExitCode)
		}
		if pipeline != nil && run.Err == nil {
			slog.Debug("Running verification pipeline", "eval", eval.Name, "treatment", t.Name, "sample", i+1)
			input := verifier.VerifyInput{
				RunOutput: run.Text,
				ExitCode:  run.ExitCode,
			}
			pr := pipeline.Run(ctx, input)
			run.Pass = &pr.Pass
			for _, step := range pr.Steps {
				slog.Debug("Verifier result", "step", step.Name, "pass", step.Result.Pass, "reason", step.Result.Reason)
			}
		}
		prog.sampleDone(run.CostUSD, run.Pass)
		tr.Runs = append(tr.Runs, run)
	}

	tr.Aggregate = result.ComputeAggregate(tr.Runs)

	return tr
}

func executeSingleRun(ctx context.Context, eval *suite.Eval, t *suite.Treatment, sample int, runner agentrunner.Runner) result.RunResult {
	opts, err := buildRunOptions(eval, t)
	if err != nil {
		return result.RunResult{
			Sample: sample,
			Err:    err,
		}
	}

	res, err := runner.Run(ctx, eval.Prompt, opts...)
	if err != nil {
		return result.RunResult{
			Sample: sample,
			Err:    err,
		}
	}

	return result.RunResult{
		Sample:     sample,
		Text:       res.Text,
		IsError:    res.IsError,
		ExitCode:   res.ExitCode,
		CostUSD:    res.CostUSD,
		DurationMs: res.DurationMs,
		Usage:      res.Usage,
		SessionID:  res.SessionID,
	}
}

func buildRunOptions(eval *suite.Eval, t *suite.Treatment) ([]agentrunner.Option, error) {
	var opts []agentrunner.Option

	// Model: treatment overrides eval.
	model := eval.Model
	if t.Model != "" {
		model = t.Model
	}
	if model != "" {
		opts = append(opts, agentrunner.WithModel(model))
	}

	// Working directory: treatment overrides eval.
	dir := eval.Dir
	if t.Dir != "" {
		dir = t.Dir
	}
	if dir != "" {
		opts = append(opts, agentrunner.WithWorkingDir(dir))
	}

	// Timeout from eval.
	if eval.Timeout != nil {
		opts = append(opts, agentrunner.WithTimeout(time.Duration(*eval.Timeout)*time.Second))
	}

	// Environment variables from treatment.
	if len(t.Env) > 0 {
		opts = append(opts, agentrunner.WithEnv(t.Env))
	}

	// Allowed tools from treatment.
	if len(t.AllowedTools) > 0 {
		opts = append(opts, claudecode.WithAllowedTools(t.AllowedTools...))
	}

	// Skill file as appended system prompt.
	if t.Skill != "" {
		content, err := os.ReadFile(t.Skill)
		if err != nil {
			return nil, fmt.Errorf("reading skill file %q: %w", t.Skill, err)
		}
		opts = append(opts, agentrunner.WithAppendSystemPrompt(string(content)))
	}

	// Always skip permissions for automated runs.
	opts = append(opts, agentrunner.WithSkipPermissions(true))

	return opts, nil
}

func filterEvals(evals []suite.Eval, ids []string) []suite.Eval {
	if len(ids) == 0 {
		return evals
	}
	idSet := toSet(ids)
	var filtered []suite.Eval
	for i := range evals {
		if idSet[evals[i].ID] {
			filtered = append(filtered, evals[i])
		}
	}
	return filtered
}

func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func shouldInclude(name string, filterSet map[string]bool) bool {
	if filterSet == nil {
		return true
	}
	return filterSet[name]
}
