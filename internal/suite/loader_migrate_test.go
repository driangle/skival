package suite

import (
	"os"
	"path/filepath"
	"testing"
)

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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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

func TestLoad_FileContainsPathKeptRelative(t *testing.T) {
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
	// file_contains paths stay relative — resolved at runtime against the workdir.
	if step.Path != "output.txt" {
		t.Errorf("file_contains path = %q, want %q", step.Path, "output.txt")
	}
}

func TestLoad_VerifyFileContainsPathKeptRelative(t *testing.T) {
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
	// file_contains paths stay relative — resolved at runtime against the workdir.
	if step.Path != "output.txt" {
		t.Errorf("file_contains path = %q, want %q", step.Path, "output.txt")
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

func TestLoad_StrictRejectsUnknownTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
treatments:
  - name: whoops
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for unknown top-level key")
	}
	if !contains(err.Error(), "treatments") {
		t.Errorf("expected error to name the unknown key %q, got: %v", "treatments", err)
	}
}

func TestLoad_StrictRejectsUnknownEvalKey(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    modle: "typo-model"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for unknown eval key")
	}
	if !contains(err.Error(), "modle") {
		t.Errorf("expected error to name the unknown key %q, got: %v", "modle", err)
	}
}

func TestLoad_StrictRejectsUnknownVariantKey(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
        modle: "typo-here"
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for unknown variant key")
	}
	if !contains(err.Error(), "modle") {
		t.Errorf("expected error to name the unknown key %q, got: %v", "modle", err)
	}
}

func TestLoad_StrictRejectsTypoedVariantsKey(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    varaints:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for typo'd variants key")
	}
	if !contains(err.Error(), "varaints") {
		t.Errorf("expected error to name the typo'd key %q, got: %v", "varaints", err)
	}
}

func TestLoad_StrictRejectsUnknownKeyInFileRef(t *testing.T) {
	dir := t.TempDir()

	evalsDir := filepath.Join(dir, "evals")
	if err := os.MkdirAll(evalsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, evalsDir, "my-eval.yaml", `
id: file-eval
prompt: "from file"
model: "claude-sonnet-4-6"
modle: "typo"
verify:
  - type: agent_exits_ok
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

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for unknown key in referenced eval file")
	}
	if !contains(err.Error(), "modle") {
		t.Errorf("expected error to name the unknown key %q, got: %v", "modle", err)
	}
	if !contains(err.Error(), "my-eval.yaml") {
		t.Errorf("expected error to name the offending file, got: %v", err)
	}
}

func TestLoad_StrictAllowsDeprecatedFields(t *testing.T) {
	// The still-supported deprecated fields must remain known so strict
	// decoding does not break back-compat.
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
      agent_exits_ok: true
      state:
        - url: "http://localhost:8080/health"
          method: GET
          expect: "ok"
    variants:
      - name: baseline
        allowed_tools:
          - Read
`)

	if _, err := Load(filepath.Join(dir, "suite.yaml")); err != nil {
		t.Fatalf("expected deprecated fields to remain valid under strict decoding, got: %v", err)
	}
}
