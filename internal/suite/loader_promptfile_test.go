package suite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustLoad loads a suite and fails the test on error.
func mustLoad(t *testing.T, dir string) *Suite {
	t.Helper()
	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return s
}

func TestLoad_PromptFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "prompt.md", "Do the thing carefully.")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt_file: prompt.md
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	s := mustLoad(t, dir)
	if got := s.Evals[0].Prompt; got != "Do the thing carefully." {
		t.Errorf("eval prompt = %q, want file contents", got)
	}
	if got := s.Evals[0].Variants[0].Prompt; got != "Do the thing carefully." {
		t.Errorf("variant prompt = %q, want inherited file contents", got)
	}
}

func TestLoad_PromptFile_Missing(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt_file: nope.md
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}
	if !strings.Contains(err.Error(), "nope.md") {
		t.Errorf("error should name the missing file, got: %v", err)
	}
}

func TestLoad_PromptFile_BothPromptAndFileOnEval(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "prompt.md", "from file")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt: "inline"
    prompt_file: prompt.md
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil || !strings.Contains(err.Error(), "both prompt and prompt_file") {
		t.Fatalf("expected both-set error, got: %v", err)
	}
}

func TestLoad_PromptFile_BothPromptAndFileOnVariant(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "v.md", "from file")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt: "eval-level"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
        prompt: "inline"
        prompt_file: v.md
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil || !strings.Contains(err.Error(), "both prompt and prompt_file") {
		t.Fatalf("expected both-set error, got: %v", err)
	}
}

func TestLoad_PromptFile_VariantOverridesEval(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "eval.md", "eval template")
	writeSuiteFile(t, dir, "variant.md", "variant template")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt_file: eval.md
    verify:
      - type: agent_exits_ok
    variants:
      - name: inherits
      - name: overrides
        prompt_file: variant.md
`)

	s := mustLoad(t, dir)
	if got := s.Evals[0].Variants[0].Prompt; got != "eval template" {
		t.Errorf("inheriting variant prompt = %q, want eval template", got)
	}
	if got := s.Evals[0].Variants[1].Prompt; got != "variant template" {
		t.Errorf("overriding variant prompt = %q, want variant template", got)
	}
}

func TestLoad_PromptFile_ResolvesRelativeToEvalFile(t *testing.T) {
	dir := t.TempDir()
	evalsDir := filepath.Join(dir, "evals")
	if err := os.MkdirAll(evalsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The prompt file lives next to the eval file, not the suite file.
	writeSuiteFile(t, evalsDir, "prompt.md", "next to the eval file")
	writeSuiteFile(t, evalsDir, "my-eval.yaml", `
id: file-eval
prompt_file: prompt.md
model: "claude-sonnet-4-6"
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

	s := mustLoad(t, dir)
	if got := s.Evals[0].Prompt; got != "next to the eval file" {
		t.Errorf("prompt = %q, want file resolved relative to eval file", got)
	}
}

func TestLoad_PromptFile_VarSubstitution(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "refactor.md", "Refactor the {{language}} code, {{tone}}.")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: refactor
    prompt_file: refactor.md
    vars:
      language: "Go"
      tone: "briefly"
    verify:
      - type: agent_exits_ok
    variants:
      - name: strict
        vars:
          tone: "terse and precise"
      - name: inherits
`)

	s := mustLoad(t, dir)
	if got := s.Evals[0].Variants[0].Prompt; got != "Refactor the Go code, terse and precise." {
		t.Errorf("strict variant prompt = %q, want variant vars to override", got)
	}
	if got := s.Evals[0].Variants[1].Prompt; got != "Refactor the Go code, briefly." {
		t.Errorf("inheriting variant prompt = %q, want eval vars", got)
	}
}

func TestLoad_PromptFile_UnresolvedPlaceholderFails(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "prompt.md", "Use {{language}} in a {{styel}} way.")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt_file: prompt.md
    vars:
      language: "Go"
      style: "clean"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	_, err := Load(filepath.Join(dir, "suite.yaml"))
	if err == nil || !strings.Contains(err.Error(), "{{styel}}") {
		t.Fatalf("expected unresolved-placeholder error naming {{styel}}, got: %v", err)
	}
}

func TestLoad_PromptFile_NoVarsPreservesLiteralBraces(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "prompt.md", "Emit JSON like {{not_a_var}} verbatim.")
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt_file: prompt.md
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	s := mustLoad(t, dir)
	if got := s.Evals[0].Variants[0].Prompt; got != "Emit JSON like {{not_a_var}} verbatim." {
		t.Errorf("prompt = %q, want literal braces preserved when no vars set", got)
	}
}

func TestLoad_PromptFile_InlinePromptUnaffectedBySubstitution(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
evals:
  - id: eval-1
    prompt: "Keep {{this}} literal."
    vars:
      this: "ignored"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	s := mustLoad(t, dir)
	if got := s.Evals[0].Prompt; got != "Keep {{this}} literal." {
		t.Errorf("inline prompt = %q, want no substitution on inline prompts", got)
	}
}
