package suite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MatrixExpansion(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
evals:
  - id: eval-1
    prompt: "compare runners"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
  - id: eval-2
    prompt: "another thing"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
  - id: eval-3
    prompt: "yet another thing"
    model: "claude-sonnet-4-6"
    isolate: false
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].Isolate == nil || !*s.Evals[0].Isolate {
		t.Error("expected eval-1 isolate to be true (explicit)")
	}
	if s.Evals[1].Isolate == nil || !*s.Evals[1].Isolate {
		t.Error("expected eval-2 isolate to be true (default)")
	}
	if s.Evals[2].Isolate == nil || *s.Evals[2].Isolate {
		t.Error("expected eval-3 isolate to be false (explicit)")
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
    verify:
      - type: agent_exits_ok
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

func TestLoad_VariantPrompt(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
evals:
  - id: eval-1
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
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
