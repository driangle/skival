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
    verify:
      - type: agent_exits_ok
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

func TestLoad_ModelPropagatesEvalToVariant(t *testing.T) {
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
      - name: override
        model: "claude-opus-4-6"
`)

	s, err := Load(filepath.Join(dir, "suite.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Evals[0].Variants[0].Model != "claude-sonnet-4-6" {
		t.Errorf("expected baseline model inherited from eval, got %q", s.Evals[0].Variants[0].Model)
	}
	if s.Evals[0].Variants[1].Model != "claude-opus-4-6" {
		t.Errorf("expected override model preserved, got %q", s.Evals[0].Variants[1].Model)
	}
}

func TestLoad_ModelPropagatesDefaultsToVariant(t *testing.T) {
	dir := t.TempDir()
	writeSuiteFile(t, dir, "suite.yaml", `
version: 1
defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
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

	if s.Evals[0].Variants[0].Model != "claude-sonnet-4-6" {
		t.Errorf("expected model propagated from defaults through eval to variant, got %q", s.Evals[0].Variants[0].Model)
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: check_output
        run: "./verify.sh"
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

	expectedVariantDir := filepath.Join(dir, "skills")
	if e.Variants[1].Dir != expectedVariantDir {
		t.Errorf("expected variant dir %q, got %q", expectedVariantDir, e.Variants[1].Dir)
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
    verify:
      - type: check_output
        run: "%s"
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
		t.Errorf("expected variant dir %q preserved, got %q", absDir, e.Variants[1].Dir)
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
