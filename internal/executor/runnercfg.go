package executor

import (
	"log/slog"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/agentrunner/go/ollama"
	execrunner "github.com/driangle/skival/internal/runners/exec"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/toolaudit"
)

// buildRunnerSpecificOpts dispatches to per-runner option builders based on runner name.
//
// claude-code is always dispatched, even with no runner_config: it applies the
// hermetic default-deny tool baseline (see buildClaudeCodeOpts), so a variant that
// declares no allowed_tools denies every built-in rather than silently inheriting
// the CLI's full tool set. Other runners have no such baseline and no-op on empty
// config.
func buildRunnerSpecificOpts(runner string, config map[string]any) []agentrunner.Option {
	switch runner {
	case "claude-code":
		return buildClaudeCodeOpts(config)
	case "ollama":
		if len(config) == 0 {
			return nil
		}
		return buildOllamaOpts(config)
	case "exec":
		if len(config) == 0 {
			return nil
		}
		return buildExecOpts(config)
	default:
		for key := range config {
			slog.Warn("Unknown runner_config key ignored", "runner", runner, "key", key)
		}
		return nil
	}
}

// buildClaudeCodeOpts maps runner_config keys to claude-code options. It always
// emits the exclusive --tools whitelist derived from allowed_tools, so the tool
// posture is deny-by-default: with allowed_tools unset, BuiltinWhitelist returns
// [""] and the runner emits --tools "" to deny every built-in (the hermetic
// baseline). Declare allowed_tools: [default] to opt back into the CLI's full
// built-in set. See docs/specs/tool-deny-enforcement.md.
func buildClaudeCodeOpts(config map[string]any) []agentrunner.Option {
	known := map[string]bool{"allowed_tools": true, "disallowed_tools": true, "mcp_config": true, "max_budget_usd": true}
	var opts []agentrunner.Option

	allowed := toStringSlice(config["allowed_tools"])
	// --tools is the enforcement lever and is always emitted (empty allow list ->
	// --tools "" -> deny all built-ins), so new built-ins never leak past an unset
	// allow list.
	opts = append(opts, claudecode.WithTools(toolaudit.BuiltinWhitelist(allowed)...))
	if len(allowed) > 0 {
		// --allowedTools preserves scoped-permission semantics for declared tools
		// and is a best-effort fallback for runners/CLI versions that ignore
		// --tools; it is not the enforcement mechanism.
		opts = append(opts, claudecode.WithAllowedTools(allowed...))
	}
	if tools := toStringSlice(config["disallowed_tools"]); len(tools) > 0 {
		// Advisory only: retained as a best-effort narrowing/fallback flag, never
		// relied on for enforcement. skival keeps no hardcoded complement of
		// built-ins, so nothing goes stale as new built-ins ship.
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

// buildExecOpts converts exec runner_config into an option carrying the parsed
// Config. Parse failures are logged and drop the config; suite validation
// already reports these to the user before a run reaches this point.
func buildExecOpts(config map[string]any) []agentrunner.Option {
	cfg, err := execrunner.ParseConfig(config)
	if err != nil {
		slog.Error("Invalid exec runner_config", "err", err)
		return nil
	}
	return []agentrunner.Option{execrunner.WithConfig(cfg)}
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
