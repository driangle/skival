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
	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/agentrunner/go/ollama"
	"github.com/driangle/skival/internal/registry"
	"github.com/driangle/skival/internal/suite"
)

// fakeRegistry creates a registry that returns the given runner for "claude-code".
func fakeRegistry(runner agentrunner.Runner) *registry.Registry {
	reg := registry.New()
	reg.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
		return runner, nil
	})
	return reg
}

// fakeRunner records calls and returns canned results.
type fakeRunner struct {
	calls    []fakeCall
	results  []*agentrunner.Result
	errs     []error
	messages [][]agentrunner.Message // per-call messages to emit
	callIdx  int
}

type fakeCall struct {
	Prompt string
	Opts   agentrunner.Options
}

func (f *fakeRunner) Run(_ context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	var o agentrunner.Options
	for _, opt := range opts {
		opt(&o)
	}
	f.calls = append(f.calls, fakeCall{Prompt: prompt, Opts: o})

	idx := f.callIdx
	f.callIdx++

	var res *agentrunner.Result
	if idx < len(f.results) {
		res = f.results[idx]
	}
	var err error
	if idx < len(f.errs) {
		err = f.errs[idx]
	}
	return res, err
}

func (f *fakeRunner) Start(_ context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Session, error) {
	var o agentrunner.Options
	for _, opt := range opts {
		opt(&o)
	}
	f.calls = append(f.calls, fakeCall{Prompt: prompt, Opts: o})

	idx := f.callIdx
	f.callIdx++

	var err error
	if idx < len(f.errs) {
		err = f.errs[idx]
	}
	if err != nil {
		return nil, err
	}

	var res *agentrunner.Result
	if idx < len(f.results) {
		res = f.results[idx]
	}

	var msgs []agentrunner.Message
	if idx < len(f.messages) {
		msgs = f.messages[idx]
	}

	ctx, cancel := context.WithCancel(context.Background())
	return agentrunner.NewSession(ctx, cancel, func(_ context.Context, ch chan<- agentrunner.Message) (*agentrunner.Result, error) {
		for _, m := range msgs {
			ch <- m
		}
		return res, nil
	}), nil
}

func intPtr(n int) *int { return &n }

func newMinimalSuite() *suite.Suite {
	return &suite.Suite{
		Description: "test suite",
		Evals: []suite.Eval{
			{
				ID:     "eval-1",
				Name:   "Test Eval",
				Prompt: "do something",
				Treatments: suite.Treatments{
					Control: suite.Treatment{Name: "control"},
				},
			},
		},
	}
}

func TestSingleEvalTreatmentSample(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "done", CostUSD: 0.05, Duration: 1500 * time.Millisecond, SessionID: "sess-1"},
		},
	}

	sr, err := Execute(context.Background(), newMinimalSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sr.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(sr.Evals))
	}
	if sr.Evals[0].EvalID != "eval-1" {
		t.Errorf("expected eval ID 'eval-1', got %q", sr.Evals[0].EvalID)
	}
	if len(sr.Evals[0].Treatments) != 1 {
		t.Fatalf("expected 1 treatment, got %d", len(sr.Evals[0].Treatments))
	}

	tr := sr.Evals[0].Treatments[0]
	if tr.Name != "control" {
		t.Errorf("expected treatment name 'control', got %q", tr.Name)
	}
	if !tr.IsControl {
		t.Error("expected IsControl to be true")
	}
	if len(tr.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(tr.Runs))
	}

	run := tr.Runs[0]
	if run.Sample != 1 {
		t.Errorf("expected sample 1, got %d", run.Sample)
	}
	if run.Text != "done" {
		t.Errorf("expected text 'done', got %q", run.Text)
	}
	if run.CostUSD != 0.05 {
		t.Errorf("expected cost 0.05, got %f", run.CostUSD)
	}
	if run.SessionID != "sess-1" {
		t.Errorf("expected session ID 'sess-1', got %q", run.SessionID)
	}
	if run.Err != nil {
		t.Errorf("expected no error, got %v", run.Err)
	}
}

func TestControlBeforeVariations(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "control-result"},
			{Text: "variation-result"},
		},
	}

	s := newMinimalSuite()
	s.Evals[0].Treatments.Variations = []suite.Treatment{
		{Name: "variation-1"},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	treatments := sr.Evals[0].Treatments
	if len(treatments) != 2 {
		t.Fatalf("expected 2 treatments, got %d", len(treatments))
	}
	if treatments[0].Name != "control" || !treatments[0].IsControl {
		t.Errorf("first treatment should be control, got %q (isControl=%v)", treatments[0].Name, treatments[0].IsControl)
	}
	if treatments[1].Name != "variation-1" || treatments[1].IsControl {
		t.Errorf("second treatment should be variation-1, got %q (isControl=%v)", treatments[1].Name, treatments[1].IsControl)
	}
	if treatments[0].Runs[0].Text != "control-result" {
		t.Error("control should run first")
	}
	if treatments[1].Runs[0].Text != "variation-result" {
		t.Error("variation should run second")
	}
}

func TestMultipleSamples(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "run-1"},
			{Text: "run-2"},
			{Text: "run-3"},
		},
	}

	s := newMinimalSuite()
	s.Evals[0].Samples = intPtr(3)

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runs := sr.Evals[0].Treatments[0].Runs
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
	for i, run := range runs {
		if run.Sample != i+1 {
			t.Errorf("run %d: expected sample %d, got %d", i, i+1, run.Sample)
		}
	}
}

func TestRunnerErrorCaptured(t *testing.T) {
	runErr := errors.New("runner exploded")
	runner := &fakeRunner{
		errs: []error{runErr},
	}

	sr, err := Execute(context.Background(), newMinimalSuite(), fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("suite should not fail on runner error, got: %v", err)
	}

	run := sr.Evals[0].Treatments[0].Runs[0]
	if run.Err == nil {
		t.Fatal("expected error in RunResult")
	}
	if run.Err.Error() != "runner exploded" {
		t.Errorf("expected 'runner exploded', got %q", run.Err.Error())
	}
}

func TestOptionsMapping(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Model = "claude-sonnet-4-6"
	s.Evals[0].Dir = "/tmp/eval-dir"
	s.Evals[0].Timeout = intPtr(30)
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:         "control",
		Env:          map[string]string{"FOO": "bar"},
		RunnerConfig: map[string]any{"allowed_tools": []string{"Read", "Write"}},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}

	opts := runner.calls[0].Opts
	if opts.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model 'claude-sonnet-4-6', got %q", opts.Model)
	}
	if opts.WorkingDir != "/tmp/eval-dir" {
		t.Errorf("expected dir '/tmp/eval-dir', got %q", opts.WorkingDir)
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", opts.Timeout)
	}
	if opts.Env["FOO"] != "bar" {
		t.Errorf("expected env FOO=bar, got %v", opts.Env)
	}
	if !opts.SkipPermissions {
		t.Error("expected SkipPermissions to be true")
	}
}

func TestTreatmentModelOverridesEval(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Model = "claude-sonnet-4-6"
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:  "control",
		Model: "claude-opus-4-6",
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if runner.calls[0].Opts.Model != "claude-opus-4-6" {
		t.Errorf("treatment model should override eval model, got %q", runner.calls[0].Opts.Model)
	}
}

func TestFilterEvals(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "result-1"},
		},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{
			{ID: "e1", Name: "Eval 1", Prompt: "p1", Treatments: suite.Treatments{Control: suite.Treatment{Name: "ctrl"}}},
			{ID: "e2", Name: "Eval 2", Prompt: "p2", Treatments: suite.Treatments{Control: suite.Treatment{Name: "ctrl"}}},
			{ID: "e3", Name: "Eval 3", Prompt: "p3", Treatments: suite.Treatments{Control: suite.Treatment{Name: "ctrl"}}},
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

func TestFilterTreatments(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "var-result"},
		},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{
			{
				ID: "e1", Name: "Eval 1", Prompt: "p1",
				Treatments: suite.Treatments{
					Control:    suite.Treatment{Name: "control"},
					Variations: []suite.Treatment{{Name: "with-skill"}},
				},
			},
		},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{Treatments: []string{"with-skill"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	treatments := sr.Evals[0].Treatments
	if len(treatments) != 1 {
		t.Fatalf("expected 1 treatment, got %d", len(treatments))
	}
	if treatments[0].Name != "with-skill" {
		t.Errorf("expected 'with-skill', got %q", treatments[0].Name)
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
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:  "control",
		Skill: skillPath,
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
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:   "control",
		Skills: []string{skillA, skillB},
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
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:   "control",
		Skills: []string{skillPath},
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
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:   "control",
		Skills: []string{skillA, "/nonexistent/b.md"},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("suite should not abort, got: %v", err)
	}

	run := sr.Evals[0].Treatments[0].Runs[0]
	if run.Err == nil {
		t.Fatal("expected error for missing skill file in skills array")
	}
}

func TestSkillFileMissing(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:  "control",
		Skill: "/nonexistent/skill.md",
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("suite should not abort, got: %v", err)
	}

	run := sr.Evals[0].Treatments[0].Runs[0]
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

	runs := sr.Evals[0].Treatments[0].Runs
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
	s.Evals[0].Model = "claude-sonnet-4-6"

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), &Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Samples from YAML should apply.
	if len(sr.Evals[0].Treatments[0].Runs) != 2 {
		t.Fatalf("expected 2 runs (YAML), got %d", len(sr.Evals[0].Treatments[0].Runs))
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

	run := sr.Evals[0].Treatments[0].Runs[0]
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

	run := sr.Evals[0].Treatments[0].Runs[0]
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
	s.Evals[0].Treatments.Control = suite.Treatment{
		Name:   "control",
		Runner: "nonexistent",
	}

	sr, err := Execute(context.Background(), s, reg, nil)
	if err != nil {
		t.Fatalf("suite should not abort: %v", err)
	}

	run := sr.Evals[0].Treatments[0].Runs[0]
	if run.Err == nil {
		t.Fatal("expected error for unknown runner")
	}
	if !strings.Contains(run.Err.Error(), "nonexistent") {
		t.Errorf("error should mention runner name, got: %v", run.Err)
	}
}

func TestTreatmentSpecificRunner(t *testing.T) {
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
	s.Evals[0].Treatments.Control = suite.Treatment{Name: "control"} // defaults to claude-code
	s.Evals[0].Treatments.Variations = []suite.Treatment{
		{Name: "ollama-variant", Runner: "ollama"},
	}

	sr, err := Execute(context.Background(), s, reg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	treatments := sr.Evals[0].Treatments
	if len(treatments) != 2 {
		t.Fatalf("expected 2 treatments, got %d", len(treatments))
	}
	if treatments[0].Runs[0].Text != "claude-result" {
		t.Errorf("control should use claude-code runner, got %q", treatments[0].Runs[0].Text)
	}
	if treatments[1].Runs[0].Text != "ollama-result" {
		t.Errorf("variation should use ollama runner, got %q", treatments[1].Runs[0].Text)
	}
}

func TestRunnerCachedAcrossTreatments(t *testing.T) {
	var createCount int
	reg := registry.New()
	reg.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
		createCount++
		return &fakeRunner{
			results: []*agentrunner.Result{{Text: "ok"}, {Text: "ok"}},
		}, nil
	})

	s := newMinimalSuite()
	s.Evals[0].Treatments.Control = suite.Treatment{Name: "control"}
	s.Evals[0].Treatments.Variations = []suite.Treatment{
		{Name: "variation"}, // also defaults to claude-code
	}

	_, err := Execute(context.Background(), s, reg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createCount != 1 {
		t.Errorf("expected runner factory called once (cached), got %d", createCount)
	}
}

func TestBuildClaudeCodeOpts(t *testing.T) {
	config := map[string]any{
		"allowed_tools":    []string{"Read", "Write"},
		"disallowed_tools": []string{"Bash"},
		"mcp_config":       "/path/to/mcp.json",
		"max_budget_usd":   1.5,
	}

	opts := buildRunnerSpecificOpts("claude-code", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	cOpts := claudecode.GetClaudeOptions(&resolved)
	if cOpts == nil {
		t.Fatal("expected claude options to be set")
	}
	if len(cOpts.AllowedTools) != 2 || cOpts.AllowedTools[0] != "Read" || cOpts.AllowedTools[1] != "Write" {
		t.Errorf("expected allowed_tools [Read Write], got %v", cOpts.AllowedTools)
	}
	if len(cOpts.DisallowedTools) != 1 || cOpts.DisallowedTools[0] != "Bash" {
		t.Errorf("expected disallowed_tools [Bash], got %v", cOpts.DisallowedTools)
	}
	if cOpts.MCPConfig != "/path/to/mcp.json" {
		t.Errorf("expected mcp_config '/path/to/mcp.json', got %q", cOpts.MCPConfig)
	}
	if cOpts.MaxBudgetUSD != 1.5 {
		t.Errorf("expected max_budget_usd 1.5, got %f", cOpts.MaxBudgetUSD)
	}
}

func TestBuildClaudeCodeOptsAnySlice(t *testing.T) {
	// YAML unmarshals string lists as []any, not []string.
	config := map[string]any{
		"allowed_tools": []any{"Read", "Write"},
	}

	opts := buildRunnerSpecificOpts("claude-code", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	cOpts := claudecode.GetClaudeOptions(&resolved)
	if cOpts == nil {
		t.Fatal("expected claude options to be set")
	}
	if len(cOpts.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(cOpts.AllowedTools))
	}
}

func TestBuildOllamaOpts(t *testing.T) {
	config := map[string]any{
		"temperature": 0.7,
		"num_ctx":     4096,
		"num_predict": 512,
		"top_p":       0.9,
		"top_k":       40,
		"seed":        42,
		"stop":        []string{"<|end|>"},
		"think":       true,
	}

	opts := buildRunnerSpecificOpts("ollama", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	oOpts := ollama.GetOllamaOptions(&resolved)
	if oOpts == nil {
		t.Fatal("expected ollama options to be set")
	}
	if oOpts.Temperature == nil || *oOpts.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", oOpts.Temperature)
	}
	if oOpts.NumCtx != 4096 {
		t.Errorf("expected num_ctx 4096, got %d", oOpts.NumCtx)
	}
	if oOpts.NumPredict != 512 {
		t.Errorf("expected num_predict 512, got %d", oOpts.NumPredict)
	}
	if oOpts.TopP == nil || *oOpts.TopP != 0.9 {
		t.Errorf("expected top_p 0.9, got %v", oOpts.TopP)
	}
	if oOpts.TopK != 40 {
		t.Errorf("expected top_k 40, got %d", oOpts.TopK)
	}
	if oOpts.Seed != 42 {
		t.Errorf("expected seed 42, got %d", oOpts.Seed)
	}
	if len(oOpts.Stop) != 1 || oOpts.Stop[0] != "<|end|>" {
		t.Errorf("expected stop [<|end|>], got %v", oOpts.Stop)
	}
	if oOpts.Think == nil || *oOpts.Think != true {
		t.Errorf("expected think true, got %v", oOpts.Think)
	}
}

func TestBuildOllamaOptsIntFromYAML(t *testing.T) {
	// YAML may unmarshal integers as int, not float64.
	config := map[string]any{
		"num_ctx": int(2048),
		"seed":    int(7),
	}

	opts := buildRunnerSpecificOpts("ollama", config)

	var resolved agentrunner.Options
	for _, o := range opts {
		o(&resolved)
	}

	oOpts := ollama.GetOllamaOptions(&resolved)
	if oOpts == nil {
		t.Fatal("expected ollama options to be set")
	}
	if oOpts.NumCtx != 2048 {
		t.Errorf("expected num_ctx 2048, got %d", oOpts.NumCtx)
	}
	if oOpts.Seed != 7 {
		t.Errorf("expected seed 7, got %d", oOpts.Seed)
	}
}

func TestBuildRunnerSpecificOptsNilConfig(t *testing.T) {
	opts := buildRunnerSpecificOpts("claude-code", nil)
	if len(opts) != 0 {
		t.Errorf("expected no options for nil config, got %d", len(opts))
	}
}

func TestBuildRunnerSpecificOptsEmptyConfig(t *testing.T) {
	opts := buildRunnerSpecificOpts("claude-code", map[string]any{})
	if len(opts) != 0 {
		t.Errorf("expected no options for empty config, got %d", len(opts))
	}
}
