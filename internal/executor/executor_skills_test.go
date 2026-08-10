package executor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/suite"
)

func TestFilterEvals(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "result-1"},
		},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{
			{ID: "e1", Name: "Eval 1", Prompt: "p1", Variants: []suite.Variant{{Name: "ctrl", Runner: "claude-code"}}},
			{ID: "e2", Name: "Eval 2", Prompt: "p2", Variants: []suite.Variant{{Name: "ctrl", Runner: "claude-code"}}},
			{ID: "e3", Name: "Eval 3", Prompt: "p3", Variants: []suite.Variant{{Name: "ctrl", Runner: "claude-code"}}},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{EvalIDs: []string{"e2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sr.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(sr.Evals))
	}
	if sr.Evals[0].EvalID != "e2" {
		t.Errorf("expected eval e2, got %q", sr.Evals[0].EvalID)
	}
}

func TestFilterVariants(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "var-result"},
		},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{
			{
				ID: "e1", Name: "Eval 1", Prompt: "p1",
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code"},
					{Name: "with-skill", Runner: "claude-code"},
				},
			},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{Variants: []string{"with-skill"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	variants := sr.Evals[0].Variants
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	if variants[0].Name != "with-skill" {
		t.Errorf("expected 'with-skill', got %q", variants[0].Name)
	}
}

func TestSkillFilePassedAsSystemPrompt(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "skill.md")
	if err := os.WriteFile(skillPath, []byte("You are a helpful assistant."), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "claude-code",
			Skill:  skillPath,
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if runner.calls[0].Opts.AppendSystemPrompt != "You are a helpful assistant." {
		t.Errorf("expected skill content as system prompt, got %q", runner.calls[0].Opts.AppendSystemPrompt)
	}
}

func TestSkillsArrayConcatenated(t *testing.T) {
	dir := t.TempDir()
	skillA := filepath.Join(dir, "a.md")
	skillB := filepath.Join(dir, "b.md")
	if err := os.WriteFile(skillA, []byte("Skill A content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillB, []byte("Skill B content"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "claude-code",
			Skills: []string{skillA, skillB},
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	expected := "Skill A content\n\nSkill B content"
	if runner.calls[0].Opts.AppendSystemPrompt != expected {
		t.Errorf("expected concatenated skills %q, got %q", expected, runner.calls[0].Opts.AppendSystemPrompt)
	}
}

func TestSkillsArraySingleFile(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "skill.md")
	if err := os.WriteFile(skillPath, []byte("Single skill"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "claude-code",
			Skills: []string{skillPath},
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if runner.calls[0].Opts.AppendSystemPrompt != "Single skill" {
		t.Errorf("expected single skill content, got %q", runner.calls[0].Opts.AppendSystemPrompt)
	}
}

func TestSkillsArrayMissingFile(t *testing.T) {
	dir := t.TempDir()
	skillA := filepath.Join(dir, "a.md")
	if err := os.WriteFile(skillA, []byte("Skill A"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "claude-code",
			Skills: []string{skillA, "/nonexistent/b.md"},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("suite should not abort, got: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if run.Err == nil {
		t.Fatal("expected error for missing skill file in skills array")
	}
}

func TestSkillFileMissing(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "claude-code",
			Skill:  "/nonexistent/skill.md",
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("suite should not abort, got: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if run.Err == nil {
		t.Fatal("expected error for missing skill file")
	}
}

func TestSamplesOverride(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "r1"}, {Text: "r2"}, {Text: "r3"}, {Text: "r4"}, {Text: "r5"},
		},
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(2) // YAML says 2

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{Samples: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Variants[0].Runs
	if len(runs) != 5 {
		t.Fatalf("expected 5 runs (CLI override), got %d", len(runs))
	}
}

func TestNoOverrideUsesYAML(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "r1"}, {Text: "r2"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(2)
	s.Evals[0].Variants[0].Model = "claude-sonnet-4-6"

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Samples from YAML should apply.
	if len(sr.Evals[0].Variants[0].Runs) != 2 {
		t.Fatalf("expected 2 runs (YAML), got %d", len(sr.Evals[0].Variants[0].Runs))
	}
	// Model from YAML should apply.
	if runner.calls[0].Opts.Model != "claude-sonnet-4-6" {
		t.Errorf("expected YAML model 'claude-sonnet-4-6', got %q", runner.calls[0].Opts.Model)
	}
}

func TestConversationPopulated(t *testing.T) {
	msg := agentrunner.Message{Raw: json.RawMessage(`{"role":"assistant","text":"done"}`)}
	runner := &fakeRunner{
		results:  []*agentrunner.Result{{Text: "done", CostUSD: 0.05, Duration: time.Second}},
		messages: [][]agentrunner.Message{{msg}},
	}

	sr, err := Execute(context.Background(), newMinimalSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if len(run.Conversation) != 1 {
		t.Fatalf("expected 1 conversation message, got %d", len(run.Conversation))
	}
	if string(run.Conversation[0]) != `{"role":"assistant","text":"done"}` {
		t.Errorf("unexpected conversation content: %s", run.Conversation[0])
	}
}

func TestConversationNilOnError(t *testing.T) {
	runner := &fakeRunner{
		errs: []error{errors.New("boom")},
	}

	sr, err := Execute(context.Background(), newMinimalSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if run.Conversation != nil {
		t.Error("expected nil conversation on error")
	}
}

func TestUnknownRunnerCapturedAsError(t *testing.T) {
	reg := registry.New()
	reg.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
		return &fakeRunner{results: []*agentrunner.Result{{Text: "ok"}}}, nil
	})

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "nonexistent",
		},
	}

	sr, err := Execute(context.Background(), s, reg, nil)
	if err != nil {
		t.Fatalf("suite should not abort: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if run.Err == nil {
		t.Fatal("expected error for unknown runner")
	}
	if !strings.Contains(run.Err.Error(), "nonexistent") {
		t.Errorf("error should mention runner name, got: %v", run.Err)
	}
}

func TestVariantSpecificRunner(t *testing.T) {
	claudeRunner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "claude-result"}},
	}
	ollamaRunner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ollama-result"}},
	}

	reg := registry.New()
	reg.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
		return claudeRunner, nil
	})
	reg.Register("ollama", func(config map[string]any) (agentrunner.Runner, error) {
		return ollamaRunner, nil
	})

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{Name: "control", Runner: "claude-code"},
		{Name: "ollama-variant", Runner: "ollama"},
	}

	sr, err := Execute(context.Background(), s, reg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	variants := sr.Evals[0].Variants
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if variants[0].Runs[0].Text != "claude-result" {
		t.Errorf("control should use claude-code runner, got %q", variants[0].Runs[0].Text)
	}
	if variants[1].Runs[0].Text != "ollama-result" {
		t.Errorf("variation should use ollama runner, got %q", variants[1].Runs[0].Text)
	}
}

func TestRunnerCachedAcrossVariants(t *testing.T) {
	var createCount int
	reg := registry.New()
	reg.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
		createCount++
		return &fakeRunner{
			results: []*agentrunner.Result{{Text: "ok"}, {Text: "ok"}},
		}, nil
	})

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{Name: "control", Runner: "claude-code"},
		{Name: "variation", Runner: "claude-code"},
	}

	_, err := Execute(context.Background(), s, reg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createCount != 1 {
		t.Errorf("expected runner factory called once (cached), got %d", createCount)
	}
}
