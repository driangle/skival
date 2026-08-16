package executor

import (
	"context"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/agentrunner/go/ollama"
	"github.com/driangle/skival/internal/suite"
)

func TestBuildClaudeCodeOpts(t *testing.T) {
	config := map[string]any{
		"allowed_tools":    []string{"Read", "Write"},
		"disallowed_tools": []string{"Bash"},
		"mcp_config":       "/path/to/mcp.json",
		"max_budget_usd":   1.5,
	}

	opts := buildRunnerSpecificOpts("claude-code", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	cOpts := claudecode.GetClaudeOptions(&resolved)
	if cOpts == nil {
		t.Fatal("expected claude options to be set")
	}
	if len(cOpts.AllowedTools) != 2 || cOpts.AllowedTools[0] != "Read" || cOpts.AllowedTools[1] != "Write" {
		t.Errorf("expected allowed_tools [Read Write], got %v", cOpts.AllowedTools)
	}
	if len(cOpts.DisallowedTools) != 1 || cOpts.DisallowedTools[0] != "Bash" {
		t.Errorf("expected disallowed_tools [Bash], got %v", cOpts.DisallowedTools)
	}
	// allowed_tools also drives --tools as an exclusive built-in whitelist.
	if len(cOpts.Tools) != 2 || cOpts.Tools[0] != "Read" || cOpts.Tools[1] != "Write" {
		t.Errorf("expected tools whitelist [Read Write], got %v", cOpts.Tools)
	}
	if cOpts.MCPConfig != "/path/to/mcp.json" {
		t.Errorf("expected mcp_config '/path/to/mcp.json', got %q", cOpts.MCPConfig)
	}
	if cOpts.MaxBudgetUSD != 1.5 {
		t.Errorf("expected max_budget_usd 1.5, got %f", cOpts.MaxBudgetUSD)
	}
}

func TestBuildClaudeCodeOptsAnySlice(t *testing.T) {
	// YAML unmarshals string lists as []any, not []string.
	config := map[string]any{
		"allowed_tools": []any{"Read", "Write"},
	}

	opts := buildRunnerSpecificOpts("claude-code", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	cOpts := claudecode.GetClaudeOptions(&resolved)
	if cOpts == nil {
		t.Fatal("expected claude options to be set")
	}
	if len(cOpts.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(cOpts.AllowedTools))
	}
}

func TestBuildClaudeCodeOptsToolsWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		allowed []any
		want    []string
	}{
		{"strips scope suffix", []any{"Bash(git:*)", "Read"}, []string{"Bash", "Read"}},
		{"drops mcp, keeps built-in", []any{"Read", "mcp__foo__bar"}, []string{"Read"}},
		{"only mcp disables all built-ins", []any{"mcp__foo__bar"}, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := buildRunnerSpecificOpts("claude-code", map[string]any{"allowed_tools": tt.allowed})
			var resolved agentrunner.Options
			for _, o := range opts {
				o(&resolved)
			}
			cOpts := claudecode.GetClaudeOptions(&resolved)
			if cOpts == nil {
				t.Fatal("expected claude options to be set")
			}
			if len(cOpts.Tools) != len(tt.want) {
				t.Fatalf("tools = %v, want %v", cOpts.Tools, tt.want)
			}
			for i, w := range tt.want {
				if cOpts.Tools[i] != w {
					t.Fatalf("tools = %v, want %v", cOpts.Tools, tt.want)
				}
			}
		})
	}
}

func TestBuildOllamaOpts(t *testing.T) {
	config := map[string]any{
		"temperature": 0.7,
		"num_ctx":     4096,
		"num_predict": 512,
		"top_p":       0.9,
		"top_k":       40,
		"seed":        42,
		"stop":        []string{"<|end|>"},
		"think":       true,
	}

	opts := buildRunnerSpecificOpts("ollama", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	oOpts := ollama.GetOllamaOptions(&resolved)
	if oOpts == nil {
		t.Fatal("expected ollama options to be set")
	}
	if oOpts.Temperature == nil || *oOpts.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", oOpts.Temperature)
	}
	if oOpts.NumCtx != 4096 {
		t.Errorf("expected num_ctx 4096, got %d", oOpts.NumCtx)
	}
	if oOpts.NumPredict != 512 {
		t.Errorf("expected num_predict 512, got %d", oOpts.NumPredict)
	}
	if oOpts.TopP == nil || *oOpts.TopP != 0.9 {
		t.Errorf("expected top_p 0.9, got %v", oOpts.TopP)
	}
	if oOpts.TopK != 40 {
		t.Errorf("expected top_k 40, got %d", oOpts.TopK)
	}
	if oOpts.Seed != 42 {
		t.Errorf("expected seed 42, got %d", oOpts.Seed)
	}
	if len(oOpts.Stop) != 1 || oOpts.Stop[0] != "<|end|>" {
		t.Errorf("expected stop [<|end|>], got %v", oOpts.Stop)
	}
	if oOpts.Think == nil || *oOpts.Think != true {
		t.Errorf("expected think true, got %v", oOpts.Think)
	}
}

func TestBuildOllamaOptsIntFromYAML(t *testing.T) {
	// YAML may unmarshal integers as int, not float64.
	config := map[string]any{
		"num_ctx": int(2048),
		"seed":    int(7),
	}

	opts := buildRunnerSpecificOpts("ollama", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	oOpts := ollama.GetOllamaOptions(&resolved)
	if oOpts == nil {
		t.Fatal("expected ollama options to be set")
	}
	if oOpts.NumCtx != 2048 {
		t.Errorf("expected num_ctx 2048, got %d", oOpts.NumCtx)
	}
	if oOpts.Seed != 7 {
		t.Errorf("expected seed 7, got %d", oOpts.Seed)
	}
}

func TestBuildRunnerSpecificOptsNilConfig(t *testing.T) {
	opts := buildRunnerSpecificOpts("claude-code", nil)
	if len(opts) != 0 {
		t.Errorf("expected no options for nil config, got %d", len(opts))
	}
}

func TestBuildRunnerSpecificOptsEmptyConfig(t *testing.T) {
	opts := buildRunnerSpecificOpts("claude-code", map[string]any{})
	if len(opts) != 0 {
		t.Errorf("expected no options for empty config, got %d", len(opts))
	}
}
func TestVariantPromptOverridesEval(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}, {Text: "ok"}},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{
			{
				ID:     "e1",
				Name:   "Eval",
				Prompt: "eval prompt",
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code", Prompt: "variant prompt"},
					{Name: "uses-eval-prompt", Runner: "claude-code"},
				},
			},
		},
	}

	_, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(runner.calls))
	}
	if runner.calls[0].Prompt != "variant prompt" {
		t.Errorf("control should use variant prompt, got %q", runner.calls[0].Prompt)
	}
	if runner.calls[1].Prompt != "eval prompt" {
		t.Errorf("variation should fall back to eval prompt, got %q", runner.calls[1].Prompt)
	}
}

func TestConfigDirSetsEnvVar(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:      "control",
			Runner:    "claude-code",
			ConfigDir: "/tmp/custom-claude-config",
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if runner.calls[0].Opts.Env["CLAUDE_CONFIG_DIR"] != "/tmp/custom-claude-config" {
		t.Errorf("expected CLAUDE_CONFIG_DIR='/tmp/custom-claude-config', got %q", runner.calls[0].Opts.Env["CLAUDE_CONFIG_DIR"])
	}
}

func TestConfigDirMergesWithExistingEnv(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:      "control",
			Runner:    "claude-code",
			ConfigDir: "/tmp/config",
			Env:       map[string]string{"FOO": "bar"},
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	opts := runner.calls[0].Opts
	if opts.Env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %q", opts.Env["FOO"])
	}
	if opts.Env["CLAUDE_CONFIG_DIR"] != "/tmp/config" {
		t.Errorf("expected CLAUDE_CONFIG_DIR='/tmp/config', got %q", opts.Env["CLAUDE_CONFIG_DIR"])
	}
}

func TestConfigDirDoesNotMutateOriginalEnv(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	originalEnv := map[string]string{"FOO": "bar"}
	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:      "control",
			Runner:    "claude-code",
			ConfigDir: "/tmp/config",
			Env:       originalEnv,
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if _, ok := originalEnv["CLAUDE_CONFIG_DIR"]; ok {
		t.Error("original env map should not be mutated")
	}
}
