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
func buildRunnerSpecificOpts(runner string, config map[string]any) []agentrunner.Option {
	if len(config) == 0 {
		return nil
	}

	switch runner {
	case "claude-code":
		return buildClaudeCodeOpts(config)
	case "ollama":
		return buildOllamaOpts(config)
	case "exec":
		return buildExecOpts(config)
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
		// --allowedTools preserves scoped-permission semantics; --tools makes the
		// allow list an exclusive whitelist that denies unlisted built-ins
		// (see docs/specs/tool-deny-enforcement.md).
		opts = append(opts, claudecode.WithAllowedTools(tools...))
		opts = append(opts, claudecode.WithTools(toolaudit.BuiltinWhitelist(tools)...))
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
