package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCmd_ValidSuite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "suite.yaml")
	content := `
version: 1
description: "test suite"
defaults:
  runner: claude-code
evals:
  - id: eval-1
    name: "Test Eval"
    prompt: "do something"
    model: "claude-sonnet-4-6"
    complexity: medium
    treatments:
      control:
        name: "baseline"
      variations:
        - name: "variant-1"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out := captureValidateOutput(t, path)

	assertContains(t, out, "OK")
	assertContains(t, out, "version:     1")
	assertContains(t, out, "description: test suite")
	assertContains(t, out, "evals:       1")
	assertContains(t, out, `eval "eval-1"`)
	assertContains(t, out, `name:       Test Eval`)
	assertContains(t, out, `complexity: medium`)
	assertContains(t, out, `treatments: 2`)
	assertContains(t, out, `"baseline"`)
	assertContains(t, out, `"variant-1"`)
}

func TestValidateCmd_InvalidSuite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := `
version: 0
evals: []
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := validateCmd
	cmd.SetArgs([]string{path})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{path})
	if err == nil {
		t.Fatal("expected error for invalid suite")
	}

	errOut := stderr.String()
	assertContains(t, errOut, "FAIL")
}

func TestValidateCmd_FileNotFound(t *testing.T) {
	cmd := validateCmd
	err := cmd.RunE(cmd, []string{"/nonexistent/suite.yaml"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidateCmd_VerifierCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "suite.yaml")
	content := `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "do something"
    model: "claude-sonnet-4-6"
    correctness:
      agent_exits_ok: true
      expected_output:
        - "hello"
      script: "./verify.sh"
    treatments:
      control:
        name: "baseline"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out := captureValidateOutput(t, path)
	assertContains(t, out, "verifiers:  3")
}

func captureValidateOutput(t *testing.T, path string) string {
	t.Helper()
	cmd := validateCmd
	cmd.SetArgs([]string{path})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return stdout.String()
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}
