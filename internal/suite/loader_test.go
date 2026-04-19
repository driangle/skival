package suite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MinimalSuite(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "do the thing"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Version != 1 {
		t.Errorf("expected version 1, got %d", s.Version)
	}
	if len(s.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(s.Evals))
	}
	if s.Evals[0].ID != "eval-1" {
		t.Errorf("expected eval ID %q, got %q", "eval-1", s.Evals[0].ID)
	}
}

func TestLoad_FileReference(t *testing.T) {
	dir := t.TempDir()

	// Create the eval file in a subdirectory
	evalsDir := filepath.Join(dir, "evals")
	if err := os.MkdirAll(evalsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, evalsDir, "my-eval.yaml", `
id: file-eval
prompt: "from file"
model: "claude-sonnet-4-6"
variants:
  - name: baseline
`)

	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - file: evals/my-eval.yaml
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(s.Evals))
	}
	if s.Evals[0].ID != "file-eval" {
		t.Errorf("expected eval ID %q, got %q", "file-eval", s.Evals[0].ID)
	}
	if s.Evals[0].File != "" {
		t.Error("expected File field to be cleared after resolution")
	}
}

func TestLoad_MissingFileReference(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - file: nonexistent.yaml
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file reference")
	}
}

func TestLoad_DefaultsMerge(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  samples: 5
  timeout: 120
  model: "claude-sonnet"
evals:
  - id: eval-1
    prompt: "task"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Samples == nil || *e.Samples != 5 {
		t.Errorf("expected samples=5 from defaults, got %v", e.Samples)
	}
	if e.Timeout == nil || *e.Timeout != 120 {
		t.Errorf("expected timeout=120 from defaults, got %v", e.Timeout)
	}
	if e.Model != "claude-sonnet" {
		t.Errorf("expected model=%q from defaults, got %q", "claude-sonnet", e.Model)
	}
}

func TestLoad_EvalOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  samples: 5
  timeout: 120
  model: "claude-sonnet"
evals:
  - id: eval-1
    prompt: "task"
    samples: 10
    timeout: 30
    model: "claude-opus"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Samples == nil || *e.Samples != 10 {
		t.Errorf("expected samples=10, got %v", e.Samples)
	}
	if e.Timeout == nil || *e.Timeout != 30 {
		t.Errorf("expected timeout=30, got %v", e.Timeout)
	}
	if e.Model != "claude-opus" {
		t.Errorf("expected model=%q, got %q", "claude-opus", e.Model)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals: [[[invalid
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 0
evals: []
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected validation error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestLoad_MissingSuiteFile(t *testing.T) {
	_, err := Load("/nonexistent/path/suite.yaml")
	if err == nil {
		t.Fatal("expected error for missing suite file")
	}
}

func TestLoad_ResolvesEvalDirToSuiteDir(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].Dir != dir {
		t.Errorf("expected eval dir to default to suite dir %q, got %q", dir, s.Evals[0].Dir)
	}
}

func TestLoad_ResolvesRelativeEvalDir(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "workdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    dir: workdir
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].Dir != subDir {
		t.Errorf("expected eval dir %q, got %q", subDir, s.Evals[0].Dir)
	}
}

func TestLoad_ResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, skillsDir, "my-skill.md", "skill content")
	writeSuiteFile(t, dir, "verify.sh", "#!/bin/bash\nexit 0")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    correctness:
      check_output: "./verify.sh"
    variants:
      - name: baseline
      - name: with-skill
        skill: "./skills/my-skill.md"
        dir: "skills"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	expectedCheckOutput := filepath.Join(dir, "verify.sh")
	checkOutputStep := findVerifyStep(e.Verify, "check_output")
	if checkOutputStep == nil {
		t.Fatal("expected check_output verify step")
	}
	if checkOutputStep.Run != expectedCheckOutput {
		t.Errorf("expected script path %q, got %q", expectedCheckOutput, checkOutputStep.Run)
	}

	expectedSkill := filepath.Join(dir, "skills", "my-skill.md")
	if e.Variants[1].Skill != expectedSkill {
		t.Errorf("expected skill path %q, got %q", expectedSkill, e.Variants[1].Skill)
	}

	expectedTreatDir := filepath.Join(dir, "skills")
	if e.Variants[1].Dir != expectedTreatDir {
		t.Errorf("expected treatment dir %q, got %q", expectedTreatDir, e.Variants[1].Dir)
	}
}

func TestLoad_PreservesAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	absDir := "/absolute/path"
	absCheckOutput := "/absolute/verify.sh"
	absSkill := "/absolute/skill.md"

	writeSuiteFile(t, dir, "suite.yaml", fmt.Sprintf(`
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    dir: "%s"
    correctness:
      check_output: "%s"
    variants:
      - name: baseline
      - name: v1
        skill: "%s"
        dir: "%s"
`, absDir, absCheckOutput, absSkill, absDir))

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Dir != absDir {
		t.Errorf("expected eval dir %q preserved, got %q", absDir, e.Dir)
	}
	absCheckOutputStep := findVerifyStep(e.Verify, "check_output")
	if absCheckOutputStep == nil {
		t.Fatal("expected check_output verify step")
	}
	if absCheckOutputStep.Run != absCheckOutput {
		t.Errorf("expected script %q preserved, got %q", absCheckOutput, absCheckOutputStep.Run)
	}
	if e.Variants[1].Skill != absSkill {
		t.Errorf("expected skill %q preserved, got %q", absSkill, e.Variants[1].Skill)
	}
	if e.Variants[1].Dir != absDir {
		t.Errorf("expected treatment dir %q preserved, got %q", absDir, e.Variants[1].Dir)
	}
}

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
		t.Errorf("expected treatment runner %q, got %q", "aider", ctrl.Runner)
	}
	if ctrl.RunnerConfig["edit_format"] != "diff" {
		t.Errorf("expected treatment runner_config.edit_format=%q, got %v", "diff", ctrl.RunnerConfig["edit_format"])
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

func TestLoad_MigrateAllowedToolsToRunnerConfig(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
        allowed_tools:
          - Read
          - Write
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Variants[0]
	if ctrl.AllowedTools != nil {
		t.Error("expected AllowedTools to be nil after migration")
	}
	if ctrl.RunnerConfig == nil {
		t.Fatal("expected RunnerConfig to be populated after migration")
	}
	tools, ok := ctrl.RunnerConfig["allowed_tools"]
	if !ok {
		t.Fatal("expected runner_config.allowed_tools to be set")
	}
	toolSlice, ok := tools.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", tools)
	}
	if len(toolSlice) != 2 || toolSlice[0] != "Read" || toolSlice[1] != "Write" {
		t.Errorf("expected [Read Write], got %v", toolSlice)
	}
}

func TestLoad_MigrateAllowedToolsDoesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
        allowed_tools:
          - Read
        runner_config:
          allowed_tools:
            - Write
            - Edit
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Variants[0]
	tools := ctrl.RunnerConfig["allowed_tools"]
	// The explicit runner_config value should win over the deprecated field.
	toolSlice, ok := tools.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", tools)
	}
	if len(toolSlice) != 2 {
		t.Errorf("expected 2 tools from runner_config, got %v", toolSlice)
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

func TestLoad_RunnerPropagatesEvalToTreatment(t *testing.T) {
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

func TestLoad_TreatmentOverridesEvalRunner(t *testing.T) {
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
	// Treatment runner is not overwritten
	if ctrl.Runner != "aider" {
		t.Errorf("expected control runner %q (treatment override), got %q", "aider", ctrl.Runner)
	}
	// Treatment key overrides eval key
	if ctrl.RunnerConfig["sandbox"] != "none" {
		t.Errorf("expected sandbox=%q (treatment override), got %v", "none", ctrl.RunnerConfig["sandbox"])
	}
	// Treatment-only key preserved
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
		t.Errorf("variation max_turns: want 20 (treatment), got %v", v.RunnerConfig["max_turns"])
	}
	if v.RunnerConfig["custom_flag"] != true {
		t.Errorf("variation custom_flag: want true (treatment), got %v", v.RunnerConfig["custom_flag"])
	}
	if v.RunnerConfig["verbose"] != true {
		t.Errorf("variation verbose: want true (defaults), got %v", v.RunnerConfig["verbose"])
	}
	if v.RunnerConfig["timeout"] != 30 {
		t.Errorf("variation timeout: want 30 (eval), got %v", v.RunnerConfig["timeout"])
	}
}

func TestLoad_MatrixExpansion(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: eval-1
    prompt: "compare runners"
    model: "claude-sonnet-4-6"
    matrix:
      dimensions:
        - name: runner
          values:
            - label: claude-code
              runner: claude-code
            - label: ollama
              runner: ollama
        - name: model
          values:
            - label: opus
              model: claude-opus-4-6
            - label: sonnet
              model: claude-sonnet-4-6
`)
	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	// 2x2 = 4 variants: first is control
	if len(e.Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(e.Variants))
	}
	if e.Variants[0].Name != "claude-code_opus" {
		t.Errorf("variant[0] name = %q, want %q", e.Variants[0].Name, "claude-code_opus")
	}
	if e.Variants[1].Name != "claude-code_sonnet" {
		t.Errorf("variant[1] name = %q, want %q", e.Variants[1].Name, "claude-code_sonnet")
	}
}

func TestLoad_MatrixSingleDimension(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "compare models"
    matrix:
      dimensions:
        - name: model
          values:
            - label: opus
              model: claude-opus-4-6
            - label: sonnet
              model: claude-sonnet-4-6
`)
	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if len(e.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(e.Variants))
	}
	if e.Variants[0].Name != "opus" {
		t.Errorf("variant[0] name = %q, want %q", e.Variants[0].Name, "opus")
	}
	if e.Variants[0].Model != "claude-opus-4-6" {
		t.Errorf("variant[0] model = %q, want %q", e.Variants[0].Model, "claude-opus-4-6")
	}
	if e.Variants[1].Name != "sonnet" {
		t.Errorf("variant[1] name = %q, want %q", e.Variants[1].Name, "sonnet")
	}
}

func TestLoad_MatrixAndVariantsMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "conflict"
    model: "claude-sonnet-4-6"
    matrix:
      dimensions:
        - name: model
          values:
            - label: opus
              model: claude-opus-4-6
    variants:
      - name: manual
`)
	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for matrix+variants, got nil")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	found := false
	for _, e := range ve.Errors {
		if contains(e, "cannot define both matrix and variants") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about matrix/variants conflict, got: %v", ve.Errors)
	}
}

func TestLoad_MatrixDimensionValues(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: eval-1
    prompt: "test"
    model: "claude-sonnet-4-6"
    matrix:
      dimensions:
        - name: runner
          values:
            - label: claude-code
              runner: claude-code
        - name: skill
          values:
            - label: baseline
`)
	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Variants[0]
	if ctrl.DimensionValues["runner"] != "claude-code" {
		t.Errorf("expected dimension runner=%q, got %q", "claude-code", ctrl.DimensionValues["runner"])
	}
	if ctrl.DimensionValues["skill"] != "baseline" {
		t.Errorf("expected dimension skill=%q, got %q", "baseline", ctrl.DimensionValues["skill"])
	}
}

func TestLoad_ResolvesSkillsPaths(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, skillsDir, "a.md", "skill A")
	writeSuiteFile(t, skillsDir, "b.md", "skill B")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
      - name: with-skills
        skills:
          - "./skills/a.md"
          - "./skills/b.md"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v := s.Evals[0].Variants[1]
	expectedA := filepath.Join(dir, "skills", "a.md")
	expectedB := filepath.Join(dir, "skills", "b.md")
	if len(v.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(v.Skills))
	}
	if v.Skills[0] != expectedA {
		t.Errorf("expected skills[0] %q, got %q", expectedA, v.Skills[0])
	}
	if v.Skills[1] != expectedB {
		t.Errorf("expected skills[1] %q, got %q", expectedB, v.Skills[1])
	}
}

func TestLoad_SkillsPreservesAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
      - name: v1
        skills:
          - "/absolute/skill-a.md"
          - "/absolute/skill-b.md"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v := s.Evals[0].Variants[1]
	if v.Skills[0] != "/absolute/skill-a.md" {
		t.Errorf("expected absolute path preserved, got %q", v.Skills[0])
	}
	if v.Skills[1] != "/absolute/skill-b.md" {
		t.Errorf("expected absolute path preserved, got %q", v.Skills[1])
	}
}

func TestLoad_SkillAndSkillsMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
        skill: "a.md"
        skills:
          - "b.md"
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected validation error for skill+skills, got nil")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	found := false
	for _, e := range ve.Errors {
		if contains(e, "cannot set both skill and skills") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about skill/skills conflict, got: %v", ve.Errors)
	}
}

func TestLoad_IsolateField(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "do the thing"
    model: "claude-sonnet-4-6"
    isolate: true
    variants:
      - name: baseline
  - id: eval-2
    prompt: "another thing"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.Evals[0].Isolate {
		t.Error("expected eval-1 isolate to be true")
	}
	if s.Evals[1].Isolate {
		t.Error("expected eval-2 isolate to be false (default)")
	}
}

func TestLoad_ResolvesConfigDirPath(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "configs", "strict")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
        config_dir: "./configs/strict"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, "configs", "strict")
	if s.Evals[0].Variants[0].ConfigDir != expected {
		t.Errorf("expected config_dir %q, got %q", expected, s.Evals[0].Variants[0].ConfigDir)
	}
}

func TestLoad_TreatmentPrompt(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    model: "claude-sonnet-4-6"
    variants:
      - name: ctrl
        prompt: "do A"
      - name: v1
        prompt: "do B"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].Variants[0].Prompt != "do A" {
		t.Errorf("expected control prompt %q, got %q", "do A", s.Evals[0].Variants[0].Prompt)
	}
	if s.Evals[0].Variants[1].Prompt != "do B" {
		t.Errorf("expected variation prompt %q, got %q", "do B", s.Evals[0].Variants[1].Prompt)
	}
}

func TestLoad_Examples(t *testing.T) {
	// Smoke test: every examples/<name>/suite.yaml must load without errors.
	root, err := filepath.Abs(filepath.Join("..", "..", "examples"))
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading examples dir: %v", err)
	}

	var found int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		suitePath := filepath.Join(root, e.Name(), "suite.yaml")
		if _, err := os.Stat(suitePath); err != nil {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			if _, err := Load(suitePath); err != nil {
				t.Errorf("Load(%s/suite.yaml) failed: %v", name, err)
			}
		})
		found++
	}

	if found == 0 {
		t.Fatal("no example suite.yaml files found")
	}
}

func TestLoad_RetryConfigParsing(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  retry:
    max_attempts: 3
    backoff: exponential
    delay: 1s
    on: transient
evals:
  - id: eval-1
    prompt: "task"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := s.Defaults.Retry
	if r == nil {
		t.Fatal("expected defaults retry to be set")
	}
	if *r.MaxAttempts != 3 {
		t.Errorf("expected max_attempts=3, got %d", *r.MaxAttempts)
	}
	if r.Backoff != "exponential" {
		t.Errorf("expected backoff=exponential, got %q", r.Backoff)
	}
	if r.Delay != "1s" {
		t.Errorf("expected delay=1s, got %q", r.Delay)
	}
	if r.On != "transient" {
		t.Errorf("expected on=transient, got %q", r.On)
	}
}

func TestLoad_RetryInheritsPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  retry:
    max_attempts: 2
    on: transient
evals:
  - id: eval-1
    prompt: "task"
    retry:
      max_attempts: 3
      on: all
    variants:
      - name: ctrl
      - name: v1
        retry:
          max_attempts: 5
  - id: eval-2
    prompt: "task2"
    variants:
      - name: ctrl2
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// eval-1 has its own retry, overrides defaults
	e1 := s.Evals[0]
	if e1.Retry == nil || *e1.Retry.MaxAttempts != 3 {
		t.Errorf("eval-1 retry should be 3 (eval override)")
	}
	if e1.Retry.On != "all" {
		t.Errorf("eval-1 retry.on should be all")
	}

	// control inherits from eval (eval > defaults)
	ctrl := e1.Variants[0]
	if ctrl.Retry == nil || *ctrl.Retry.MaxAttempts != 3 {
		t.Errorf("control should inherit retry from eval (3)")
	}

	// v1 has its own retry, overrides eval
	v1 := e1.Variants[1]
	if v1.Retry == nil || *v1.Retry.MaxAttempts != 5 {
		t.Errorf("v1 should have retry max_attempts=5 (treatment override)")
	}

	// eval-2 inherits from defaults
	e2 := s.Evals[1]
	if e2.Retry == nil || *e2.Retry.MaxAttempts != 2 {
		t.Errorf("eval-2 should inherit retry from defaults (2)")
	}

	// eval-2 control inherits from eval (which inherited from defaults)
	ctrl2 := e2.Variants[0]
	if ctrl2.Retry == nil || *ctrl2.Retry.MaxAttempts != 2 {
		t.Errorf("ctrl2 should inherit retry from defaults via eval (2)")
	}
}

func TestLoad_RankingWeightsParsing(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
ranking:
  weights:
    correctness: 0.50
    cost: 0.30
    duration: 0.20
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Ranking == nil {
		t.Fatal("expected ranking config to be set")
	}
	if s.Ranking.Weights.Correctness != 0.50 {
		t.Errorf("expected correctness=0.50, got %g", s.Ranking.Weights.Correctness)
	}
	if s.Ranking.Weights.Cost != 0.30 {
		t.Errorf("expected cost=0.30, got %g", s.Ranking.Weights.Cost)
	}
	if s.Ranking.Weights.Duration != 0.20 {
		t.Errorf("expected duration=0.20, got %g", s.Ranking.Weights.Duration)
	}
}

func TestLoad_RankingWeightsOmitted(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Ranking != nil {
		t.Error("expected ranking to be nil when omitted")
	}
}

func TestLoad_RankingWeightsInvalidSum(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
ranking:
  weights:
    correctness: 0.50
    cost: 0.30
    duration: 0.30
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected validation error for weights not summing to 1.0")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	found := false
	for _, e := range ve.Errors {
		if contains(e, "must sum to 1.0") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about weights sum, got: %v", ve.Errors)
	}
}

func TestLoad_RankingWeightsNegative(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
ranking:
  weights:
    correctness: -0.10
    cost: 0.80
    duration: 0.30
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected validation error for negative weight")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	found := false
	for _, e := range ve.Errors {
		if contains(e, "must be >= 0") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about negative weight, got: %v", ve.Errors)
	}
}

func TestLoad_JudgeModelOnEval(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    correctness:
      judge: ["output is correct"]
      judge_model: "claude-opus-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	judgeStep := findJudgeStep(s.Evals[0].Verify)
	if judgeStep == nil {
		t.Fatal("expected judge step in verify")
	}
	if judgeStep.Model != "claude-opus-4-6" {
		t.Errorf("judge step model = %q, want %q", judgeStep.Model, "claude-opus-4-6")
	}
}

func TestLoad_JudgeModelFromDefaults(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  judge_model: "claude-opus-4-6"
evals:
  - id: eval-1
    prompt: "task"
    correctness:
      judge: ["output is correct"]
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	judgeStep := findJudgeStep(s.Evals[0].Verify)
	if judgeStep == nil {
		t.Fatal("expected judge step in verify")
	}
	if judgeStep.Model != "claude-opus-4-6" {
		t.Errorf("judge step model = %q, want %q", judgeStep.Model, "claude-opus-4-6")
	}
}

func TestLoad_JudgeModelEvalOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  judge_model: "claude-haiku-4-5-20251001"
evals:
  - id: eval-1
    prompt: "task"
    correctness:
      judge: ["output is correct"]
      judge_model: "claude-opus-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	judgeStep := findJudgeStep(s.Evals[0].Verify)
	if judgeStep == nil {
		t.Fatal("expected judge step in verify")
	}
	if judgeStep.Model != "claude-opus-4-6" {
		t.Errorf("judge step model = %q, want %q (eval should override defaults)", judgeStep.Model, "claude-opus-4-6")
	}
}

func TestLoad_JudgeModelOmittedUsesNoOverride(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    correctness:
      judge: ["output is correct"]
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	judgeStep := findJudgeStep(s.Evals[0].Verify)
	if judgeStep == nil {
		t.Fatal("expected judge step in verify")
	}
	if judgeStep.Model != "" {
		t.Errorf("judge step model = %q, want empty (default should be applied at runtime)", judgeStep.Model)
	}
}

func TestLoad_MigrateStateToVerifySteps(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: state-test
    prompt: "test"
    model: "claude-sonnet-4-6"
    correctness:
      state:
        - url: "http://localhost:8080/health"
          method: GET
          expect: "ok"
        - url: "http://localhost:8080/ready"
          method: POST
          expect: "ready"
    variants:
      - name: baseline
        runner: claude-code
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	v := s.Evals[0].Verify
	if len(v) != 2 {
		t.Fatalf("expected 2 verify steps from migration, got %d", len(v))
	}

	if v[0].Type != "http_check" {
		t.Errorf("step[0] type = %q, want http_check", v[0].Type)
	}
	if v[0].URL != "http://localhost:8080/health" {
		t.Errorf("step[0] URL = %q, want http://localhost:8080/health", v[0].URL)
	}
	if v[0].Method != "GET" {
		t.Errorf("step[0] Method = %q, want GET", v[0].Method)
	}
	if v[0].BodyContains != "ok" {
		t.Errorf("step[0] BodyContains = %q, want ok", v[0].BodyContains)
	}

	if v[1].Type != "http_check" {
		t.Errorf("step[1] type = %q, want http_check", v[1].Type)
	}
	if v[1].Method != "POST" {
		t.Errorf("step[1] Method = %q, want POST", v[1].Method)
	}
	if v[1].BodyContains != "ready" {
		t.Errorf("step[1] BodyContains = %q, want ready", v[1].BodyContains)
	}
}

func TestLoad_FileContainsPathResolved(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: file-contains-test
    prompt: "test"
    model: "claude-sonnet-4-6"
    correctness:
      probes:
        - file:
            path: "output.txt"
            assert:
              exists: true
    variants:
      - name: baseline
        runner: claude-code
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	step := findVerifyStep(s.Evals[0].Verify, "file_contains")
	if step == nil {
		t.Fatal("expected file_contains verify step")
	}
	expected := filepath.Join(dir, "output.txt")
	if step.Path != expected {
		t.Errorf("file_contains path = %q, want %q", step.Path, expected)
	}
}

func TestLoad_VerifyFileContainsPathResolved(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: file-contains-test
    prompt: "test"
    model: "claude-sonnet-4-6"
    verify:
      - type: file_contains
        path: "output.txt"
    variants:
      - name: baseline
        runner: claude-code
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	step := findVerifyStep(s.Evals[0].Verify, "file_contains")
	if step == nil {
		t.Fatal("expected file_contains verify step")
	}
	expected := filepath.Join(dir, "output.txt")
	if step.Path != expected {
		t.Errorf("file_contains path = %q, want %q", step.Path, expected)
	}
}

func TestLoad_MigrateCorrectnessToVerify(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: migrate-test
    prompt: "test"
    model: "claude-sonnet-4-6"
    correctness:
      agent_exits_ok: true
      check: "go build ./..."
      output:
        contains: ["hello"]
      check_output: "./verify.sh"
      judge: ["is correct"]
      judge_model: "claude-opus-4-6"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	e := s.Evals[0]
	if len(e.Verify) != 5 {
		t.Fatalf("expected 5 verify steps, got %d", len(e.Verify))
	}

	expected := []string{"agent_exits_ok", "check", "output_contains", "check_output", "judge"}
	for i, typ := range expected {
		if e.Verify[i].Type != typ {
			t.Errorf("step[%d] type = %q, want %q", i, e.Verify[i].Type, typ)
		}
	}

	judgeStep := findJudgeStep(e.Verify)
	if judgeStep == nil {
		t.Fatal("expected judge step in verify")
	}
	if judgeStep.Model != "claude-opus-4-6" {
		t.Errorf("judge step model = %q, want %q", judgeStep.Model, "claude-opus-4-6")
	}
}

func TestLoad_VerifyDirectFormat(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: verify-test
    prompt: "test"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
      - type: check
        run: "go build ./..."
      - type: output_contains
        values: ["hello"]
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	e := s.Evals[0]
	if len(e.Verify) != 3 {
		t.Fatalf("expected 3 verify steps, got %d", len(e.Verify))
	}

	if e.Verify[0].Type != "agent_exits_ok" {
		t.Errorf("step[0] type = %q, want agent_exits_ok", e.Verify[0].Type)
	}
	if e.Verify[1].Type != "check" || e.Verify[1].Run != "go build ./..." {
		t.Errorf("step[1] = %+v, want check with run", e.Verify[1])
	}
	if e.Verify[2].Type != "output_contains" || len(e.Verify[2].Values) != 1 {
		t.Errorf("step[2] = %+v, want output_contains with values", e.Verify[2])
	}
}

func findVerifyStep(steps []VerifyStep, typ string) *VerifyStep {
	for i := range steps {
		if steps[i].Type == typ {
			return &steps[i]
		}
	}
	return nil
}

func writeSuiteFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findJudgeStep(steps []VerifyStep) *VerifyStep {
	for i := range steps {
		if steps[i].Type == "judge" {
			return &steps[i]
		}
	}
	return nil
}
