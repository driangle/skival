package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/agentrunner/go/ollama"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/verifier"
)

// Execute runs all evals in the suite, returning collected results.
// Runner errors are captured per-run and do not abort the suite.

func Execute(ctx context.Context, s *suite.Suite, reg *registry.Registry, opts *Options) (*result.SuiteResult, error) {
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
		prog.evalStart(i+1, len(evals), evalLabel(&evals[i]))
		evalResult := executeEval(ctx, &evals[i], reg, opts, prog)
		sr.Evals = append(sr.Evals, evalResult)
	}

	sr.FinishedAt = time.Now()
	prog.finish()
	return sr, nil
}

func executeEval(ctx context.Context, eval *suite.Eval, reg *registry.Registry, opts *Options, prog *progress) result.EvalResult {
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
		er.Err = err
		for _, v := range variants {
			er.Skipped = append(er.Skipped, result.SkippedVariant{
				Name:   v.variant.Name,
				Reason: "before hook failed",
			})
		}
		prog.evalError(label, err)
		prog.skippedVariants(label, er.Skipped)
		return er
	}

	pv := opts.ParallelVariants
	if pv > len(variants) {
		pv = len(variants)
	}

	if pv > 1 {
		// Parallel variant execution with bounded concurrency.
		results := make([]result.VariantResult, len(variants))
		sem := make(chan struct{}, pv)
		var wg sync.WaitGroup

		for i := range variants {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				cache := make(map[string]agentrunner.Runner)
				v := variants[idx]
				results[idx] = executeVariant(ctx, eval, label, v.variant, v.isControl, reg, cache, opts, prog)
			}(i)
		}
		wg.Wait()
		er.Variants = append(er.Variants, results...)
	} else {
		// Sequential execution (default).
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
					return er
				}
			}

			v := variants[i]
			vr := executeVariant(ctx, eval, label, v.variant, v.isControl, reg, runnerCache, opts, prog)
			er.Variants = append(er.Variants, vr)
		}
	}

	return er
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

func executeVariant(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, isControl bool, reg *registry.Registry, runnerCache map[string]agentrunner.Runner, opts *Options, prog *progress) result.VariantResult {
	runnerName := v.Runner

	vr := result.VariantResult{
		Name:      v.Name,
		Runner:    runnerName,
		Model:     v.Model,
		IsControl: isControl,
	}

	runner, ok := runnerCache[runnerName]
	if !ok {
		var err error
		runner, err = reg.Create(runnerName, v.RunnerConfig)
		if err != nil {
			slog.Error("Failed to create runner", "runner", runnerName, "variant", v.Name, "err", err)
			vr.Runs = append(vr.Runs, result.RunResult{Sample: 1, Err: fmt.Errorf("creating runner %q: %w", runnerName, err)})
			return vr
		}
		runnerCache[runnerName] = runner
	}

	samples := 1
	if eval.Samples != nil {
		samples = *eval.Samples
	}
	if opts.Samples > 0 {
		samples = opts.Samples
	}

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

	timeoutOverride := opts.Timeout

	if parallel <= 1 {
		// Sequential execution (default).
		for i := 0; i < samples; i++ {
			run := runSample(ctx, eval, label, v, i, samples, runner, prog, timeoutOverride)
			vr.Runs = append(vr.Runs, run)
		}
	} else {
		// Parallel execution with bounded concurrency.
		runs := make([]result.RunResult, samples)
		sem := make(chan struct{}, parallel)
		var wg sync.WaitGroup

		for i := 0; i < samples; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				runs[idx] = runSample(ctx, eval, label, v, idx, samples, runner, prog, timeoutOverride)
			}(i)
		}
		wg.Wait()
		vr.Runs = runs
	}

	vr.Aggregate = result.ComputeAggregate(vr.Runs)

	return vr
}

// runSample executes a single sample, including isolation, running, verification,
// and retry logic based on the variant's retry config.
func runSample(ctx context.Context, eval *suite.Eval, label string, v *suite.Variant, idx, samples int, runner agentrunner.Runner, prog *progress, timeoutOverride int) result.RunResult {
	sample := idx + 1
	prog.sampleStart(label, v.Name, sample, samples)
	slog.Debug("Running sample", "eval", label, "variant", v.Name, "sample", sample, "total", samples)

	sampleDir := ""
	if eval.Isolate != nil && *eval.Isolate {
		var err error
		sampleDir, err = createIsolatedDir(eval, v)
		if err != nil {
			prog.sampleDone(label, v.Name, sample, 0, nil)
			return result.RunResult{Sample: sample, Err: fmt.Errorf("creating isolated dir: %w", err)}
		}
	}

	// Resolve the working directory for display.
	workdir := sampleDir
	if workdir == "" {
		if v.Dir != "" {
			workdir = v.Dir
		} else {
			workdir = eval.Dir
		}
	}
	if workdir != "" {
		prog.workdir(label, v.Name, workdir)
	}

	retryCfg := resolveRetryConfig(v.Retry)

	// Build verification pipeline once (reused across attempts).
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
	pipeline := verifier.BuildPipeline(eval.Verify, verifyDir, pipelineOpts...)

	var bestRun result.RunResult
	for attempt := 1; attempt <= retryCfg.maxAttempts; attempt++ {
		if attempt > 1 {
			delay := backoffDelay(attempt-1, retryCfg)
			slog.Debug("Retrying sample", "eval", label, "variant", v.Name, "sample", sample, "attempt", attempt, "delay", delay)
			time.Sleep(delay)
		}

		run := executeSingleRun(ctx, eval, v, sample, runner, sampleDir, timeoutOverride)
		if run.Err != nil {
			slog.Debug("Sample error", "eval", label, "variant", v.Name, "sample", sample, "attempt", attempt, "err", run.Err)
		} else {
			slog.Debug("Sample complete", "eval", label, "variant", v.Name, "sample", sample, "attempt", attempt,
				"cost", run.CostUSD, "duration_ms", run.DurationMs, "exit_code", run.ExitCode)
			prog.sessionID(label, v.Name, run.SessionID)
		}

		// Run verification if execution succeeded.
		if pipeline != nil && run.Err == nil {
			slog.Debug("Running verification pipeline", "eval", label, "variant", v.Name, "sample", sample, "attempt", attempt)
			input := verifier.VerifyInput{
				RunOutput:    run.Text,
				ExitCode:     run.ExitCode,
				Conversation: run.Conversation,
			}
			pr := pipeline.Run(ctx, input)
			run.Pass = &pr.Pass
			prog.verifyResults(label, v.Name, pr.Steps)
			for _, step := range pr.Steps {
				slog.Debug("Verifier result", "step", step.Name, "pass", step.Result.Pass, "reason", step.Result.Reason)
				if step.Name == "judge" && step.Result.Conversation != nil {
					run.JudgeConversation = step.Result.Conversation
				}
			}
		}

		// Tag retry metadata.
		run.Attempt = attempt
		run.Retried = attempt > 1

		// Keep the best result across attempts.
		if attempt == 1 || isBetterResult(run, bestRun) {
			bestRun = run
		}

		// If the run passed or shouldn't be retried, stop.
		if run.Pass != nil && *run.Pass {
			bestRun.TotalAttempts = attempt
			break
		}
		if attempt < retryCfg.maxAttempts && !shouldRetry(&run, retryCfg) {
			bestRun.TotalAttempts = attempt
			break
		}
		bestRun.TotalAttempts = attempt
	}

	bestRun.WorkDir = workdir
	prog.sampleDone(label, v.Name, sample, bestRun.CostUSD, bestRun.Pass)
	return bestRun
}

// createIsolatedDir creates a temporary directory and copies the eval/variant working directory into it.
func createIsolatedDir(eval *suite.Eval, v *suite.Variant) (string, error) {
	srcDir := eval.Dir
	if v.Dir != "" {
		srcDir = v.Dir
	}

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

func executeSingleRun(ctx context.Context, eval *suite.Eval, v *suite.Variant, sample int, runner agentrunner.Runner, isolatedDir string, timeoutOverride int) result.RunResult {
	opts, err := buildRunOptions(eval, v, isolatedDir, timeoutOverride)
	if err != nil {
		return result.RunResult{
			Sample: sample,
			Err:    err,
		}
	}

	// Prompt: variant > eval.
	prompt := eval.Prompt
	if v.Prompt != "" {
		prompt = v.Prompt
	}

	session, err := runner.Start(ctx, prompt, opts...)
	if err != nil {
		return result.RunResult{
			Sample: sample,
			Err:    err,
		}
	}

	var conversation []json.RawMessage
	for msg := range session.Messages {
		if msg.Raw != nil {
			conversation = append(conversation, msg.Raw)
		}
	}

	res, err := session.Result()
	if err != nil {
		return result.RunResult{
			Sample: sample,
			Err:    err,
		}
	}

	return result.RunResult{
		Sample:       sample,
		Text:         res.Text,
		IsError:      res.IsError,
		ExitCode:     res.ExitCode,
		CostUSD:      res.CostUSD,
		DurationMs:   res.Duration.Milliseconds(),
		Usage:        res.Usage,
		SessionID:    res.SessionID,
		Conversation: conversation,
	}
}

func buildRunOptions(eval *suite.Eval, v *suite.Variant, isolatedDir string, timeoutOverride int) ([]agentrunner.Option, error) {
	var opts []agentrunner.Option

	if v.Model != "" {
		opts = append(opts, agentrunner.WithModel(v.Model))
	}

	// Working directory: isolated > variant > eval.
	dir := eval.Dir
	if v.Dir != "" {
		dir = v.Dir
	}
	if isolatedDir != "" {
		dir = isolatedDir
	}
	if dir != "" {
		opts = append(opts, agentrunner.WithWorkingDir(dir))
	}

	// Timeout: CLI override > eval-level.
	if timeoutOverride > 0 {
		opts = append(opts, agentrunner.WithTimeout(time.Duration(timeoutOverride)*time.Second))
	} else if eval.Timeout != nil {
		opts = append(opts, agentrunner.WithTimeout(time.Duration(*eval.Timeout)*time.Second))
	}

	// Environment variables from variant, plus CLAUDE_CONFIG_DIR if config_dir is set.
	env := v.Env
	if v.ConfigDir != "" {
		if env == nil {
			env = make(map[string]string)
		} else {
			// Copy to avoid mutating the original variant.
			copied := make(map[string]string, len(env)+1)
			for k, val := range env {
				copied[k] = val
			}
			env = copied
		}
		env["CLAUDE_CONFIG_DIR"] = v.ConfigDir
	}
	if len(env) > 0 {
		opts = append(opts, agentrunner.WithEnv(env))
	}

	// Runner-specific options from runner_config.
	opts = append(opts, buildRunnerSpecificOpts(v.Runner, v.RunnerConfig)...)

	// Skill file(s) as appended system prompt.
	skillPrompt, err := loadSkillContent(v)
	if err != nil {
		return nil, err
	}
	if skillPrompt != "" {
		opts = append(opts, agentrunner.WithAppendSystemPrompt(skillPrompt))
	}

	// Always skip permissions for automated runs.
	opts = append(opts, agentrunner.WithSkipPermissions())

	return opts, nil
}

// loadSkillContent reads skill file(s) from a variant and returns the concatenated content.
// Returns empty string if no skills are configured.
func loadSkillContent(v *suite.Variant) (string, error) {
	if v.Skill != "" {
		content, err := os.ReadFile(v.Skill)
		if err != nil {
			return "", fmt.Errorf("reading skill file %q: %w", v.Skill, err)
		}
		return string(content), nil
	}

	if len(v.Skills) > 0 {
		var parts []string
		for _, path := range v.Skills {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("reading skill file %q: %w", path, err)
			}
			parts = append(parts, string(content))
		}
		return strings.Join(parts, "\n\n"), nil
	}

	return "", nil
}

// buildRunnerSpecificOpts dispatches to per-runner option builders based on runner name.
func buildRunnerSpecificOpts(runner string, config map[string]any) []agentrunner.Option {
	if len(config) == 0 {
		return nil
	}

	switch runner {
	case "claude-code":
		return buildClaudeCodeOpts(config)
	case "ollama":
		return buildOllamaOpts(config)
	default:
		for key := range config {
			slog.Warn("Unknown runner_config key ignored", "runner", runner, "key", key)
		}
		return nil
	}
}

// buildClaudeCodeOpts maps runner_config keys to claude-code options.
func buildClaudeCodeOpts(config map[string]any) []agentrunner.Option {
	known := map[string]bool{"allowed_tools": true, "disallowed_tools": true, "mcp_config": true, "max_budget_usd": true}
	var opts []agentrunner.Option

	if tools := toStringSlice(config["allowed_tools"]); len(tools) > 0 {
		opts = append(opts, claudecode.WithAllowedTools(tools...))
	}
	if tools := toStringSlice(config["disallowed_tools"]); len(tools) > 0 {
		opts = append(opts, claudecode.WithDisallowedTools(tools...))
	}
	if path, ok := config["mcp_config"].(string); ok && path != "" {
		opts = append(opts, claudecode.WithMCPConfig(path))
	}
	if budget, ok := toFloat64(config["max_budget_usd"]); ok {
		opts = append(opts, claudecode.WithMaxBudgetUSD(budget))
	}

	for key := range config {
		if !known[key] {
			slog.Warn("Unknown runner_config key for claude-code", "key", key)
		}
	}

	return opts
}

// buildOllamaOpts maps runner_config keys to ollama options.
func buildOllamaOpts(config map[string]any) []agentrunner.Option {
	known := map[string]bool{"temperature": true, "num_ctx": true, "num_predict": true, "top_p": true, "top_k": true, "seed": true, "stop": true, "think": true}
	var opts []agentrunner.Option

	if temp, ok := toFloat64(config["temperature"]); ok {
		opts = append(opts, ollama.WithTemperature(temp))
	}
	if n, ok := toInt(config["num_ctx"]); ok {
		opts = append(opts, ollama.WithNumCtx(n))
	}
	if n, ok := toInt(config["num_predict"]); ok {
		opts = append(opts, ollama.WithNumPredict(n))
	}
	if p, ok := toFloat64(config["top_p"]); ok {
		opts = append(opts, ollama.WithTopP(p))
	}
	if k, ok := toInt(config["top_k"]); ok {
		opts = append(opts, ollama.WithTopK(k))
	}
	if s, ok := toInt(config["seed"]); ok {
		opts = append(opts, ollama.WithSeed(s))
	}
	if sequences := toStringSlice(config["stop"]); len(sequences) > 0 {
		opts = append(opts, ollama.WithStop(sequences...))
	}
	if think, ok := config["think"].(bool); ok {
		opts = append(opts, ollama.WithThink(think))
	}

	for key := range config {
		if !known[key] {
			slog.Warn("Unknown runner_config key for ollama", "key", key)
		}
	}

	return opts
}

// toStringSlice converts a value to []string, handling both []string and []any.
func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		var strs []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				strs = append(strs, s)
			}
		}
		return strs
	}
	return nil
}

// toFloat64 converts a numeric value to float64.
func toFloat64(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	}
	return 0, false
}

// toInt converts a numeric value to int.
func toInt(v any) (int, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	}
	return 0, false
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

// evalLabel returns a display label for the eval, preferring Name over ID.
func evalLabel(eval *suite.Eval) string {
	if eval.Name != "" {
		return eval.Name
	}
	return eval.ID
}

func hasJudgeStep(steps []suite.VerifyStep) bool {
	for _, s := range steps {
		if s.Type == "judge" {
			return true
		}
	}
	return false
}
