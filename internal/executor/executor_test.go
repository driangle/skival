package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	agentrunner "github.com/driangle/agent-runner/go"
	"github.com/driangle/skival/internal/suite"
)

// fakeRunner records calls and returns canned results.
type fakeRunner struct {
	calls   []fakeCall
	results []*agentrunner.Result
	errs    []error
	callIdx int
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

func (f *fakeRunner) Start(_ context.Context, _ string, _ ...agentrunner.Option) *agentrunner.Session {
	return nil
}

func (f *fakeRunner) RunStream(_ context.Context, _ string, _ ...agentrunner.Option) (<-chan agentrunner.Message, <-chan error) {
	return nil, nil
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
			{Text: "done", CostUSD: 0.05, DurationMs: 1500, SessionID: "sess-1"},
		},
	}

	sr, err := Execute(context.Background(), newMinimalSuite(), runner, nil)
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

	sr, err := Execute(context.Background(), s, runner, nil)
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

	sr, err := Execute(context.Background(), s, runner, nil)
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

	sr, err := Execute(context.Background(), newMinimalSuite(), runner, nil)
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
		AllowedTools: []string{"Read", "Write"},
	}

	Execute(context.Background(), s, runner, nil)

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

	Execute(context.Background(), s, runner, nil)

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

	sr, err := Execute(context.Background(), s, runner, &Options{EvalIDs: []string{"e2"}})
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

	sr, err := Execute(context.Background(), s, runner, &Options{Treatments: []string{"with-skill"}})
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
