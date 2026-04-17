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

func writeSuiteFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
