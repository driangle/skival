package executor

import (
	"context"
	"time"

	agentrunner "github.com/driangle/agent-runner/go"
	"github.com/driangle/agent-runner/go/claudecode"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// Execute runs all evals in the suite, returning collected results.
// Runner errors are captured per-run and do not abort the suite.
func Execute(ctx context.Context, s *suite.Suite, runner agentrunner.Runner) (*result.SuiteResult, error) {
	sr := &result.SuiteResult{
		Description: s.Description,
		StartedAt:   time.Now(),
	}

	for i := range s.Evals {
		evalResult := executeEval(ctx, &s.Evals[i], runner)
		sr.Evals = append(sr.Evals, evalResult)
	}

	sr.FinishedAt = time.Now()
	return sr, nil
}

func executeEval(ctx context.Context, eval *suite.Eval, runner agentrunner.Runner) result.EvalResult {
	er := result.EvalResult{
		EvalID:   eval.ID,
		EvalName: eval.Name,
	}

	// Control first, then variations.
	control := executeTreatment(ctx, eval, &eval.Treatments.Control, true, runner)
	er.Treatments = append(er.Treatments, control)

	for i := range eval.Treatments.Variations {
		variation := executeTreatment(ctx, eval, &eval.Treatments.Variations[i], false, runner)
		er.Treatments = append(er.Treatments, variation)
	}

	return er
}

func executeTreatment(ctx context.Context, eval *suite.Eval, t *suite.Treatment, isControl bool, runner agentrunner.Runner) result.TreatmentResult {
	tr := result.TreatmentResult{
		Name:      t.Name,
		IsControl: isControl,
	}

	samples := 1
	if eval.Samples != nil {
		samples = *eval.Samples
	}

	for i := 0; i < samples; i++ {
		run := executeSingleRun(ctx, eval, t, i+1, runner)
		tr.Runs = append(tr.Runs, run)
	}

	tr.Aggregate = result.ComputeAggregate(tr.Runs)

	return tr
}

func executeSingleRun(ctx context.Context, eval *suite.Eval, t *suite.Treatment, sample int, runner agentrunner.Runner) result.RunResult {
	opts := buildRunOptions(eval, t)

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

func buildRunOptions(eval *suite.Eval, t *suite.Treatment) []agentrunner.Option {
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

	// Always skip permissions for automated runs.
	opts = append(opts, agentrunner.WithSkipPermissions(true))

	return opts
}

