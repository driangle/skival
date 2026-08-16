package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
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

func intPtr(n int) *int    { return &n }
func boolPtr(b bool) *bool { return &b }

func newMinimalSuite() *suite.Suite {
	return &suite.Suite{
		Description: "test suite",
		Evals: []suite.Eval{
			{
				ID:     "eval-1",
				Name:   "Test Eval",
				Prompt: "do something",
				Variants: []suite.Variant{
					{Name: "control", Runner: "claude-code"},
				},
			},
		},
	}
}

func TestSingleEvalVariantSample(t *testing.T) {
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
	if len(sr.Evals[0].Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(sr.Evals[0].Variants))
	}

	tr := sr.Evals[0].Variants[0]
	if tr.Name != "control" {
		t.Errorf("expected variant name 'control', got %q", tr.Name)
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
	s.Evals[0].Variants = []suite.Variant{
		{Name: "control", Runner: "claude-code"},
		{Name: "variation-1", Runner: "claude-code"},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	variants := sr.Evals[0].Variants
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if variants[0].Name != "control" || !variants[0].IsControl {
		t.Errorf("first variant should be control, got %q (isControl=%v)", variants[0].Name, variants[0].IsControl)
	}
	if variants[1].Name != "variation-1" || variants[1].IsControl {
		t.Errorf("second variant should be variation-1, got %q (isControl=%v)", variants[1].Name, variants[1].IsControl)
	}
	if variants[0].Runs[0].Text != "control-result" {
		t.Error("control should run first")
	}
	if variants[1].Runs[0].Text != "variation-result" {
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

	runs := sr.Evals[0].Variants[0].Runs
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

	run := sr.Evals[0].Variants[0].Runs[0]
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
	s.Evals[0].Dir = "/tmp/eval-dir"
	s.Evals[0].Timeout = intPtr(30)
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:         "control",
			Runner:       "claude-code",
			Model:        "claude-sonnet-4-6",
			Env:          map[string]string{"FOO": "bar"},
			RunnerConfig: map[string]any{"allowed_tools": []string{"Read", "Write"}},
		},
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
	if !opts.DangerouslySkipPermissions {
		t.Error("expected DangerouslySkipPermissions to be true")
	}
}

func TestVariantModelUsed(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{
			Name:   "control",
			Runner: "claude-code",
			Model:  "claude-opus-4-6",
		},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if runner.calls[0].Opts.Model != "claude-opus-4-6" {
		t.Errorf("expected variant model, got %q", runner.calls[0].Opts.Model)
	}
}

func TestVariantResultModelFromVariant(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants[0].Model = "claude-sonnet-4-6"

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr := sr.Evals[0].Variants[0]
	if tr.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model 'claude-sonnet-4-6', got %q", tr.Model)
	}
}

func TestVariantResultModelPerVariant(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}, {Text: "ok"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Variants = []suite.Variant{
		{Name: "control", Runner: "claude-code", Model: "claude-opus-4-6"},
		{Name: "variation", Runner: "claude-code", Model: "claude-sonnet-4-6"},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sr.Evals[0].Variants[0].Model != "claude-opus-4-6" {
		t.Errorf("control should use its own model, got %q", sr.Evals[0].Variants[0].Model)
	}
	if sr.Evals[0].Variants[1].Model != "claude-sonnet-4-6" {
		t.Errorf("variation should use its own model, got %q", sr.Evals[0].Variants[1].Model)
	}
}
