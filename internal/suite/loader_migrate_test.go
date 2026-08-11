package suite

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestLoad_StrictRejectsRemovedFields(t *testing.T) {
	// The removed deprecated fields must surface a loud, field-naming error at
	// load time rather than being silently dropped.
	cases := []struct {
		name       string
		suite      string
		wantSubstr string
	}{
		{
			name: "eval correctness block",
			suite: `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    prompt: "task"
    model: "claude-sonnet-4-6"
    correctness:
      agent_exits_ok: true
    variants:
      - name: baseline
`,
			wantSubstr: "correctness",
		},
		{
			name: "variant allowed_tools",
			suite: `
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
`,
			wantSubstr: "allowed_tools",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeSuiteFile(t, dir, "suite.yaml", tc.suite)

			_, err := Load(filepath.Join(dir, "suite.yaml"))
			if err == nil {
				t.Fatalf("expected error for removed field %q", tc.wantSubstr)
			}
			if !contains(err.Error(), tc.wantSubstr) {
				t.Errorf("expected error to name the removed key %q, got: %v", tc.wantSubstr, err)
			}
		})
	}
}
