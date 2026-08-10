package suite

import (
	"path/filepath"
	"testing"
)

func TestLoad_RunnerAndRunnerConfigFields(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    max_turns: 10
evals:
  - id: eval-1
    prompt: "task"
    runner: "codex"
    runner_config:
      sandbox: "full"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
        runner: "aider"
        runner_config:
          edit_format: "diff"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Runner != "codex" {
		t.Errorf("expected eval runner %q, got %q", "codex", e.Runner)
	}
	if e.RunnerConfig["sandbox"] != "full" {
		t.Errorf("expected eval runner_config.sandbox=%q, got %v", "full", e.RunnerConfig["sandbox"])
	}

	ctrl := e.Variants[0]
	if ctrl.Runner != "aider" {
		t.Errorf("expected variant runner %q, got %q", "aider", ctrl.Runner)
	}
	if ctrl.RunnerConfig["edit_format"] != "diff" {
		t.Errorf("expected variant runner_config.edit_format=%q, got %v", "diff", ctrl.RunnerConfig["edit_format"])
	}
}

func TestLoad_DefaultsRunnerMerge(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    max_turns: 10
evals:
  - id: eval-1
    prompt: "task"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Runner != "claude-code" {
		t.Errorf("expected runner %q from defaults, got %q", "claude-code", e.Runner)
	}
	if e.RunnerConfig["max_turns"] != 10 {
		t.Errorf("expected runner_config.max_turns=10 from defaults, got %v", e.RunnerConfig["max_turns"])
	}
}

func TestLoad_RunnerConfigDeepMergeDefaultsToEval(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    max_turns: 10
    verbose: true
evals:
  - id: eval-1
    prompt: "task"
    runner_config:
      max_turns: 20
      sandbox: "full"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Runner != "claude-code" {
		t.Errorf("expected runner %q from defaults, got %q", "claude-code", e.Runner)
	}
	// Eval override wins
	if e.RunnerConfig["max_turns"] != 20 {
		t.Errorf("expected max_turns=20 (eval override), got %v", e.RunnerConfig["max_turns"])
	}
	// Default key preserved
	if e.RunnerConfig["verbose"] != true {
		t.Errorf("expected verbose=true from defaults, got %v", e.RunnerConfig["verbose"])
	}
	// Eval-only key preserved
	if e.RunnerConfig["sandbox"] != "full" {
		t.Errorf("expected sandbox=%q from eval, got %v", "full", e.RunnerConfig["sandbox"])
	}
}

func TestLoad_RunnerPropagatesEvalToVariant(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    max_turns: 10
evals:
  - id: eval-1
    prompt: "task"
    runner_config:
      sandbox: "full"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
      - name: v1
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Variants[0]
	if ctrl.Runner != "claude-code" {
		t.Errorf("expected control runner %q inherited from eval, got %q", "claude-code", ctrl.Runner)
	}
	if ctrl.RunnerConfig["max_turns"] != 10 {
		t.Errorf("expected control max_turns=10, got %v", ctrl.RunnerConfig["max_turns"])
	}
	if ctrl.RunnerConfig["sandbox"] != "full" {
		t.Errorf("expected control sandbox=%q, got %v", "full", ctrl.RunnerConfig["sandbox"])
	}

	v1 := s.Evals[0].Variants[1]
	if v1.Runner != "claude-code" {
		t.Errorf("expected variation runner %q inherited from eval, got %q", "claude-code", v1.Runner)
	}
	if v1.RunnerConfig["max_turns"] != 10 {
		t.Errorf("expected variation max_turns=10, got %v", v1.RunnerConfig["max_turns"])
	}
}

func TestLoad_VariantOverridesEvalRunner(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    max_turns: 10
evals:
  - id: eval-1
    prompt: "task"
    runner_config:
      sandbox: "full"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
        runner: "aider"
        runner_config:
          edit_format: "diff"
          sandbox: "none"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Variants[0]
	// Variant runner is not overwritten
	if ctrl.Runner != "aider" {
		t.Errorf("expected control runner %q (variant override), got %q", "aider", ctrl.Runner)
	}
	// Variant key overrides eval key
	if ctrl.RunnerConfig["sandbox"] != "none" {
		t.Errorf("expected sandbox=%q (variant override), got %v", "none", ctrl.RunnerConfig["sandbox"])
	}
	// Variant-only key preserved
	if ctrl.RunnerConfig["edit_format"] != "diff" {
		t.Errorf("expected edit_format=%q, got %v", "diff", ctrl.RunnerConfig["edit_format"])
	}
	// Eval/default key inherited
	if ctrl.RunnerConfig["max_turns"] != 10 {
		t.Errorf("expected max_turns=10 inherited from eval, got %v", ctrl.RunnerConfig["max_turns"])
	}
}

func TestLoad_FullMergeChain(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    max_turns: 5
    verbose: true
    log_level: "info"
evals:
  - id: eval-1
    prompt: "task"
    runner_config:
      max_turns: 10
      timeout: 30
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
      - name: custom
        runner: "codex"
        runner_config:
          max_turns: 20
          custom_flag: true
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]

	// Control inherits everything from eval (which merged defaults)
	ctrl := e.Variants[0]
	if ctrl.Runner != "claude-code" {
		t.Errorf("control runner: want %q, got %q", "claude-code", ctrl.Runner)
	}
	if ctrl.RunnerConfig["max_turns"] != 10 {
		t.Errorf("control max_turns: want 10 (eval), got %v", ctrl.RunnerConfig["max_turns"])
	}
	if ctrl.RunnerConfig["verbose"] != true {
		t.Errorf("control verbose: want true (defaults), got %v", ctrl.RunnerConfig["verbose"])
	}
	if ctrl.RunnerConfig["log_level"] != "info" {
		t.Errorf("control log_level: want %q (defaults), got %v", "info", ctrl.RunnerConfig["log_level"])
	}
	if ctrl.RunnerConfig["timeout"] != 30 {
		t.Errorf("control timeout: want 30 (eval), got %v", ctrl.RunnerConfig["timeout"])
	}

	// Variation overrides runner and some config, inherits the rest
	v := e.Variants[1]
	if v.Runner != "codex" {
		t.Errorf("variation runner: want %q, got %q", "codex", v.Runner)
	}
	if v.RunnerConfig["max_turns"] != 20 {
		t.Errorf("variation max_turns: want 20 (variant), got %v", v.RunnerConfig["max_turns"])
	}
	if v.RunnerConfig["custom_flag"] != true {
		t.Errorf("variation custom_flag: want true (variant), got %v", v.RunnerConfig["custom_flag"])
	}
	if v.RunnerConfig["verbose"] != true {
		t.Errorf("variation verbose: want true (defaults), got %v", v.RunnerConfig["verbose"])
	}
	if v.RunnerConfig["timeout"] != 30 {
		t.Errorf("variation timeout: want 30 (eval), got %v", v.RunnerConfig["timeout"])
	}
}
