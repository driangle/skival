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
evals:
  - id: eval-1
    prompt: "do the thing"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
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
treatments:
  control:
    name: baseline
`)

	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
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
  samples: 5
  timeout: 120
  model: "claude-sonnet"
evals:
  - id: eval-1
    prompt: "task"
    treatments:
      control:
        name: baseline
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
  samples: 5
  timeout: 120
  model: "claude-sonnet"
evals:
  - id: eval-1
    prompt: "task"
    samples: 10
    timeout: 30
    model: "claude-opus"
    treatments:
      control:
        name: baseline
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    dir: workdir
    treatments:
      control:
        name: baseline
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    correctness:
      script: "./verify.sh"
    treatments:
      control:
        name: baseline
      variations:
        - name: with-skill
          skill: "./skills/my-skill.md"
          dir: "skills"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	expectedScript := filepath.Join(dir, "verify.sh")
	if e.Correctness.Script != expectedScript {
		t.Errorf("expected script path %q, got %q", expectedScript, e.Correctness.Script)
	}

	expectedSkill := filepath.Join(dir, "skills", "my-skill.md")
	if e.Treatments.Variations[0].Skill != expectedSkill {
		t.Errorf("expected skill path %q, got %q", expectedSkill, e.Treatments.Variations[0].Skill)
	}

	expectedTreatDir := filepath.Join(dir, "skills")
	if e.Treatments.Variations[0].Dir != expectedTreatDir {
		t.Errorf("expected treatment dir %q, got %q", expectedTreatDir, e.Treatments.Variations[0].Dir)
	}
}

func TestLoad_PreservesAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	absDir := "/absolute/path"
	absScript := "/absolute/verify.sh"
	absSkill := "/absolute/skill.md"

	writeSuiteFile(t, dir, "suite.yaml", fmt.Sprintf(`
version: 1
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    dir: "%s"
    correctness:
      script: "%s"
    treatments:
      control:
        name: baseline
      variations:
        - name: v1
          skill: "%s"
          dir: "%s"
`, absDir, absScript, absSkill, absDir))

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := s.Evals[0]
	if e.Dir != absDir {
		t.Errorf("expected eval dir %q preserved, got %q", absDir, e.Dir)
	}
	if e.Correctness.Script != absScript {
		t.Errorf("expected script %q preserved, got %q", absScript, e.Correctness.Script)
	}
	if e.Treatments.Variations[0].Skill != absSkill {
		t.Errorf("expected skill %q preserved, got %q", absSkill, e.Treatments.Variations[0].Skill)
	}
	if e.Treatments.Variations[0].Dir != absDir {
		t.Errorf("expected treatment dir %q preserved, got %q", absDir, e.Treatments.Variations[0].Dir)
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
    treatments:
      control:
        name: baseline
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

	ctrl := e.Treatments.Control
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
    treatments:
      control:
        name: baseline
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
        allowed_tools:
          - Read
          - Write
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Treatments.Control
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
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

	ctrl := s.Evals[0].Treatments.Control
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
    treatments:
      control:
        name: baseline
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
    treatments:
      control:
        name: baseline
      variations:
        - name: v1
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Treatments.Control
	if ctrl.Runner != "claude-code" {
		t.Errorf("expected control runner %q inherited from eval, got %q", "claude-code", ctrl.Runner)
	}
	if ctrl.RunnerConfig["max_turns"] != 10 {
		t.Errorf("expected control max_turns=10, got %v", ctrl.RunnerConfig["max_turns"])
	}
	if ctrl.RunnerConfig["sandbox"] != "full" {
		t.Errorf("expected control sandbox=%q, got %v", "full", ctrl.RunnerConfig["sandbox"])
	}

	v1 := s.Evals[0].Treatments.Variations[0]
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
    treatments:
      control:
        name: baseline
        runner: "aider"
        runner_config:
          edit_format: "diff"
          sandbox: "none"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctrl := s.Evals[0].Treatments.Control
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
    treatments:
      control:
        name: baseline
      variations:
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
	ctrl := e.Treatments.Control
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
	v := e.Treatments.Variations[0]
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
	// 2x2 = 4 treatments: first is control, rest are variations
	if e.Treatments.Control.Name != "claude-code_opus" {
		t.Errorf("control name = %q, want %q", e.Treatments.Control.Name, "claude-code_opus")
	}
	if len(e.Treatments.Variations) != 3 {
		t.Fatalf("expected 3 variations, got %d", len(e.Treatments.Variations))
	}
	if e.Treatments.Variations[0].Name != "claude-code_sonnet" {
		t.Errorf("variation[0] name = %q, want %q", e.Treatments.Variations[0].Name, "claude-code_sonnet")
	}
}

func TestLoad_MatrixSingleDimension(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
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
	if e.Treatments.Control.Name != "opus" {
		t.Errorf("control name = %q, want %q", e.Treatments.Control.Name, "opus")
	}
	if e.Treatments.Control.Model != "claude-opus-4-6" {
		t.Errorf("control model = %q, want %q", e.Treatments.Control.Model, "claude-opus-4-6")
	}
	if len(e.Treatments.Variations) != 1 {
		t.Fatalf("expected 1 variation, got %d", len(e.Treatments.Variations))
	}
}

func TestLoad_MatrixAndTreatmentsMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
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
    treatments:
      control:
        name: manual
`)
	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for matrix+treatments, got nil")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	found := false
	for _, e := range ve.Errors {
		if contains(e, "cannot define both matrix and treatments") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about matrix/treatments conflict, got: %v", ve.Errors)
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

	ctrl := s.Evals[0].Treatments.Control
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
      variations:
        - name: with-skills
          skills:
            - "./skills/a.md"
            - "./skills/b.md"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v := s.Evals[0].Treatments.Variations[0]
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
      variations:
        - name: v1
          skills:
            - "/absolute/skill-a.md"
            - "/absolute/skill-b.md"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v := s.Evals[0].Treatments.Variations[0]
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
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
evals:
  - id: eval-1
    prompt: "do the thing"
    model: "claude-sonnet-4-6"
    isolate: true
    treatments:
      control:
        name: baseline
  - id: eval-2
    prompt: "another thing"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
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
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
        config_dir: "./configs/strict"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, "configs", "strict")
	if s.Evals[0].Treatments.Control.ConfigDir != expected {
		t.Errorf("expected config_dir %q, got %q", expected, s.Evals[0].Treatments.Control.ConfigDir)
	}
}

func TestLoad_TreatmentPrompt(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: eval-1
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: ctrl
        prompt: "do A"
      variations:
        - name: v1
          prompt: "do B"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].Treatments.Control.Prompt != "do A" {
		t.Errorf("expected control prompt %q, got %q", "do A", s.Evals[0].Treatments.Control.Prompt)
	}
	if s.Evals[0].Treatments.Variations[0].Prompt != "do B" {
		t.Errorf("expected variation prompt %q, got %q", "do B", s.Evals[0].Treatments.Variations[0].Prompt)
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

func writeSuiteFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
