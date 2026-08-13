package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/verifier"
)

// runSample executes a single sample, including isolation, running, verification,
// and retry logic based on the variant's retry config.
func runSample(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, idx, samples int, runner agentrunner.Runner, prog *progress, timeoutOverride int) result.RunResult {
	sample := idx + 1
	prog.sampleStart(label, v.Name, sample, samples)
	slog.Debug("Running sample", "eval", label, "variant", v.Name, "sample", sample, "total", samples)

	sampleDir, err := prepareSampleDir(eval, v)
	if err != nil {
		prog.sampleDone(label, v.Name, sample, 0, nil)
		return result.RunResult{Sample: sample, Err: err}
	}

	workdir := resolveWorkdir(eval, v, sampleDir)
	if workdir != "" {
		prog.workdir(label, v.Name, workdir)
	}

	pipeline := buildSamplePipeline(eval, v, runner, sampleDir)
	bestRun := runSampleAttempts(ctx, eval, label, v, sample, runner, pipeline, sampleDir, timeoutOverride, prog)
	bestRun.WorkDir = workdir
	prog.sampleDone(label, v.Name, sample, bestRun.CostUSD, bestRun.Pass)
	return bestRun
}

// prepareSampleDir creates an isolated working directory when the eval requests
// isolation, returning an empty string otherwise. Isolation is skipped when no
// working directory is configured, since there is nothing to copy.
func prepareSampleDir(eval *suite.Eval, v *suite.Variant) (string, error) {
	if eval.Isolate == nil || !*eval.Isolate {
		return "", nil
	}
	srcDir := isolationSource(eval, v)
	if srcDir == "" {
		return "", nil
	}
	dir, err := createIsolatedDir(srcDir)
	if err != nil {
		return "", fmt.Errorf("creating isolated dir: %w", err)
	}
	return dir, nil
}

// isolationSource resolves the directory to copy when isolating: variant > eval.
func isolationSource(eval *suite.Eval, v *suite.Variant) string {
	if v.Dir != "" {
		return v.Dir
	}
	return eval.Dir
}

// resolveWorkdir resolves the working directory for display: isolated > variant > eval.
func resolveWorkdir(eval *suite.Eval, v *suite.Variant, sampleDir string) string {
	if sampleDir != "" {
		return sampleDir
	}
	if v.Dir != "" {
		return v.Dir
	}
	return eval.Dir
}

// buildSamplePipeline constructs the verification pipeline for a sample, wiring
// the judge runner and model when a judge step is present.
func buildSamplePipeline(eval *suite.Eval, v *suite.Variant, runner agentrunner.Runner, sampleDir string) *verifier.Pipeline {
	verifyDir := eval.Dir
	if sampleDir != "" {
		verifyDir = sampleDir
	}
	judgePrompt := eval.Prompt
	if v.Prompt != "" {
		judgePrompt = v.Prompt
	}
	var pipelineOpts []verifier.PipelineOption
	if hasJudgeStep(eval.Verify) {
		pipelineOpts = append(pipelineOpts,
			verifier.WithJudge(runner, judgePrompt),
			verifier.WithAgentModel(v.Model),
		)
	}
	return verifier.BuildPipeline(eval.Verify, verifyDir, eval.SuiteDir, pipelineOpts...)
}

// runSampleAttempts runs a sample with retries, keeping the best result across attempts.
func runSampleAttempts(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, sample int, runner agentrunner.Runner, pipeline *verifier.Pipeline, sampleDir string, timeoutOverride int, prog *progress) result.RunResult {
	retryCfg := resolveRetryConfig(v.Retry)

	var bestRun result.RunResult
	for attempt := 1; attempt <= retryCfg.maxAttempts; attempt++ {
		if attempt > 1 {
			delay := backoffDelay(attempt-1, retryCfg)
			slog.Debug("Retrying sample", "eval", label, "variant", v.Name, "sample", sample, "attempt", attempt, "delay", delay)
			if !sleepContext(ctx, delay) {
				break // context cancelled during backoff; return best result so far
			}
		}

		run := executeSingleRun(ctx, eval, v, sample, runner, sampleDir, timeoutOverride)
		logRunOutcome(run, label, v.Name, sample, attempt, prog)

		if pipeline != nil && run.Err == nil {
			run = runVerification(ctx, pipeline, run, label, v.Name, sample, attempt, prog)
		}

		run.Attempt = attempt
		run.Retried = attempt > 1
		if attempt == 1 || isBetterResult(run, bestRun) {
			bestRun = run
		}
		bestRun.TotalAttempts = attempt

		if attemptIsFinal(run, attempt, retryCfg) {
			break
		}
	}
	return bestRun
}

// logRunOutcome emits debug logs and progress for a single attempt's outcome.
func logRunOutcome(run result.RunResult, label, variant string, sample, attempt int, prog *progress) {
	if run.Err != nil {
		slog.Debug("Sample error", "eval", label, "variant", variant, "sample", sample, "attempt", attempt, "err", run.Err)
		return
	}
	slog.Debug("Sample complete", "eval", label, "variant", variant, "sample", sample, "attempt", attempt,
		"cost", run.CostUSD, "duration_ms", run.DurationMs, "exit_code", run.ExitCode)
	prog.sessionID(label, variant, run.SessionID)
}

// runVerification runs the verification pipeline against a completed run and
// returns the run enriched with pass state and judge conversation.
func runVerification(ctx context.Context, pipeline *verifier.Pipeline, run result.RunResult, label, variant string, sample, attempt int, prog *progress) result.RunResult {
	slog.Debug("Running verification pipeline", "eval", label, "variant", variant, "sample", sample, "attempt", attempt)
	input := verifier.VerifyInput{
		RunOutput:    run.Text,
		ExitCode:     run.ExitCode,
		Conversation: run.Conversation,
	}
	pr := pipeline.Run(ctx, input)
	run.Pass = &pr.Pass
	run.Steps = mapStepResults(pr.Steps)
	prog.verifyResults(label, variant, pr.Steps)
	for _, step := range pr.Steps {
		slog.Debug("Verifier result", "step", step.Name, "pass", step.Result.Pass, "reason", step.Result.Reason)
		if step.Name == "judge" && step.Result.Conversation != nil {
			run.JudgeConversation = step.Result.Conversation
		}
	}
	return run
}

// mapStepResults converts pipeline step results into the persisted result form,
// preserving the command detail (exit code, stdout, stderr) behind each failure.
func mapStepResults(steps []verifier.StepResult) []result.StepResult {
	if len(steps) == 0 {
		return nil
	}
	out := make([]result.StepResult, 0, len(steps))
	for _, s := range steps {
		out = append(out, result.StepResult{
			Name:     s.Name,
			Type:     s.Type,
			Pass:     s.Result.Pass,
			ExitCode: s.Result.ExitCode,
			Stdout:   s.Result.Stdout,
			Stderr:   s.Result.Stderr,
			Reason:   s.Result.Reason,
		})
	}
	return out
}

// attemptIsFinal reports whether the retry loop should stop after this attempt.
func attemptIsFinal(run result.RunResult, attempt int, cfg retryConfig) bool {
	if run.Pass != nil && *run.Pass {
		return true
	}
	if attempt < cfg.maxAttempts && !shouldRetry(&run, cfg) {
		return true
	}
	return false
}

// createIsolatedDir creates a temporary directory and copies srcDir into it.
// The temp dir is intentionally left in place after the run so users can
// inspect it (it is reported in the run's Workdirs section).
func createIsolatedDir(srcDir string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "skival-isolate-*")
	if err != nil {
		return "", err
	}

	if err := copyDir(srcDir, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}
