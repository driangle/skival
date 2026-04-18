package verifier

import (
	"context"
	"testing"

	"github.com/driangle/skival/internal/suite"
)

func boolPtr(b bool) *bool { return &b }

func TestBuildPipeline_NoConfig(t *testing.T) {
	p := BuildPipeline(suite.Correctness{}, "")
	if p != nil {
		t.Fatal("expected nil pipeline for empty correctness config")
	}
}

func TestBuildPipeline_OutputOnly(t *testing.T) {
	p := BuildPipeline(suite.Correctness{
		Output: suite.Output{Contains: []string{"hello"}},
	}, "")
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 1 || p.steps[0].name != "output" {
		t.Errorf("expected single output step, got %v", p.steps)
	}
}

func TestBuildPipeline_AllVerifiers(t *testing.T) {
	p := BuildPipeline(suite.Correctness{
		AgentExitsOK:   boolPtr(true),
		Output:         suite.Output{Contains: []string{"ok"}},
		State:          []suite.StateAssertion{{URL: "http://localhost", Expect: "up"}},
		CheckOutput:    "exit 0",
	}, "/tmp")

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(p.steps))
	}

	expected := []string{"agent_exits_ok", "output", "state", "check_output"}
	for i, name := range expected {
		if p.steps[i].name != name {
			t.Errorf("step %d: got %q, want %q", i, p.steps[i].name, name)
		}
	}
}

func TestBuildPipeline_AgentExitsOKFalseSkipped(t *testing.T) {
	p := BuildPipeline(suite.Correctness{
		AgentExitsOK: boolPtr(false),
	}, "")
	if p != nil {
		t.Fatal("expected nil pipeline when agent_exits_ok=false and nothing else configured")
	}
}

func TestPipeline_AllPass(t *testing.T) {
	p := &Pipeline{
		steps: []namedVerifier{
			{name: "a", verifier: &stubVerifier{pass: true, reason: "ok"}},
			{name: "b", verifier: &stubVerifier{pass: true, reason: "ok"}},
		},
	}
	r := p.Run(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Error("expected pipeline to pass")
	}
	if len(r.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(r.Steps))
	}
}

func TestPipeline_ShortCircuitOnFailure(t *testing.T) {
	called := false
	p := &Pipeline{
		steps: []namedVerifier{
			{name: "a", verifier: &stubVerifier{pass: true, reason: "ok"}},
			{name: "b", verifier: &stubVerifier{pass: false, reason: "bad"}},
			{name: "c", verifier: &callTracker{called: &called}},
		},
	}
	r := p.Run(context.Background(), VerifyInput{})
	if r.Pass {
		t.Error("expected pipeline to fail")
	}
	if len(r.Steps) != 2 {
		t.Errorf("expected 2 steps (short-circuited), got %d", len(r.Steps))
	}
	if r.Steps[1].Result.Reason != "bad" {
		t.Errorf("expected reason 'bad', got %q", r.Steps[1].Result.Reason)
	}
	if called {
		t.Error("step c should not have been called")
	}
}

func TestPipeline_FirstStepFails(t *testing.T) {
	p := &Pipeline{
		steps: []namedVerifier{
			{name: "a", verifier: &stubVerifier{pass: false, reason: "fail"}},
			{name: "b", verifier: &stubVerifier{pass: true, reason: "ok"}},
		},
	}
	r := p.Run(context.Background(), VerifyInput{})
	if r.Pass {
		t.Error("expected pipeline to fail")
	}
	if len(r.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(r.Steps))
	}
}

func TestPipeline_SingleStep(t *testing.T) {
	p := &Pipeline{
		steps: []namedVerifier{
			{name: "only", verifier: &stubVerifier{pass: true, reason: "good"}},
		},
	}
	r := p.Run(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Error("expected pass")
	}
	if len(r.Steps) != 1 || r.Steps[0].Name != "only" {
		t.Error("unexpected steps")
	}
}

// Test helpers

type stubVerifier struct {
	pass   bool
	reason string
}

func (s *stubVerifier) Verify(_ context.Context, _ VerifyInput) VerifyResult {
	return VerifyResult{Pass: s.pass, Reason: s.reason}
}

type callTracker struct {
	called *bool
}

func (c *callTracker) Verify(_ context.Context, _ VerifyInput) VerifyResult {
	*c.called = true
	return VerifyResult{Pass: true}
}
