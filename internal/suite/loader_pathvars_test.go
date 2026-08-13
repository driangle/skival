package suite

import (
	"path/filepath"
	"testing"
)

// TestLoad_SetsSuiteDir verifies every eval is stamped with the directory
// containing the loaded suite.yaml so it can be exposed via ${SKIVAL_SUITE_DIR}.
func TestLoad_SetsSuiteDir(t *testing.T) {
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
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].SuiteDir != dir {
		t.Errorf("expected SuiteDir %q, got %q", dir, s.Evals[0].SuiteDir)
	}
}

// TestLoad_CheckOutputVarLeftRaw verifies a check_output whose run references a
// ${SKIVAL_...} variable is NOT pre-joined against the suite dir at load time;
// it is left raw for runtime substitution.
func TestLoad_CheckOutputVarLeftRaw(t *testing.T) {
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
      - type: check_output
        run: "${SKIVAL_SUITE_DIR}/grader.sh"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	step := findVerifyStep(s.Evals[0].Verify, "check_output")
	if step == nil {
		t.Fatal("expected check_output verify step")
	}
	if step.Run != "${SKIVAL_SUITE_DIR}/grader.sh" {
		t.Errorf("expected raw run left for substitution, got %q", step.Run)
	}
}

// TestLoad_CheckOutputPlainRelativeStillJoined verifies the existing behavior
// for plain relative check_output paths (no variable) is unchanged.
func TestLoad_CheckOutputPlainRelativeStillJoined(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "verify.sh", "#!/bin/sh\nexit 0")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    verify:
      - type: check_output
        run: "./verify.sh"
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	step := findVerifyStep(s.Evals[0].Verify, "check_output")
	if step == nil {
		t.Fatal("expected check_output verify step")
	}
	want := filepath.Join(dir, "verify.sh")
	if step.Run != want {
		t.Errorf("expected joined path %q, got %q", want, step.Run)
	}
}
