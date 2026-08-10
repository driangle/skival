package suite

import (
	"errors"
	"path/filepath"
	"testing"
)

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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
    variants:
      - name: ctrl
      - name: v1
        retry:
          max_attempts: 5
  - id: eval-2
    prompt: "task2"
    verify:
      - type: agent_exits_ok
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
		t.Errorf("v1 should have retry max_attempts=5 (variant override)")
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
    verify:
      - type: agent_exits_ok
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
    verify:
      - type: agent_exits_ok
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
