package verifier

import (
	"context"
	"testing"

	"github.com/driangle/skival/internal/suite"
)

func TestBuildPipeline_NoSteps(t *testing.T) {
	p := BuildPipeline(nil, "")
	if p != nil {
		t.Fatal("expected nil pipeline for nil steps")
	}

	p = BuildPipeline([]suite.VerifyStep{}, "")
	if p != nil {
		t.Fatal("expected nil pipeline for empty steps")
	}
}

func TestBuildPipeline_OutputContainsOnly(t *testing.T) {
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "output_contains", Values: []string{"hello"}},
	}, "")
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 1 || p.steps[0].name != "output_contains" {
		t.Errorf("expected single output_contains step, got %v", p.steps)
	}
}

func TestBuildPipeline_AllVerifiers(t *testing.T) {
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "agent_exits_ok"},
		{Type: "output_contains", Values: []string{"ok"}},
		{Type: "check_output", Run: "exit 0"},
	}, "/tmp")

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(p.steps))
	}

	expected := []string{"agent_exits_ok", "output_contains", "check_output"}
	for i, name := range expected {
		if p.steps[i].name != name {
			t.Errorf("step %d: got %q, want %q", i, p.steps[i].name, name)
		}
	}
}

func TestBuildPipeline_WithTypedSteps(t *testing.T) {
	trueVal := true
	exitZero := 0
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "http_check", URL: "http://localhost", BodyContains: "ok"},
		{Type: "file_contains", Path: "/tmp/test.txt", Exists: &trueVal},
		{Type: "command", Run: "echo hi", Exits: &exitZero},
		{Type: "tcp_check", Host: "localhost", Port: 8080},
	}, "/tmp")

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(p.steps))
	}

	expected := []string{"http_check[0]", "file_contains[1]", "command[2]", "tcp_check[3]"}
	for i, name := range expected {
		if p.steps[i].name != name {
			t.Errorf("step %d: got %q, want %q", i, p.steps[i].name, name)
		}
	}
}

func TestBuildPipeline_StepsRunInListOrder(t *testing.T) {
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "check_output", Run: "exit 0"},
		{Type: "http_check", URL: "http://localhost"},
		{Type: "agent_exits_ok"},
	}, "/tmp")

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(p.steps))
	}

	expected := []string{"check_output", "http_check[1]", "agent_exits_ok"}
	for i, name := range expected {
		if p.steps[i].name != name {
			t.Errorf("step %d: got %q, want %q", i, p.steps[i].name, name)
		}
	}
}

func TestBuildPipeline_CustomName(t *testing.T) {
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "command", Name: "tests_pass", Run: "pytest tests/ -q"},
	}, "/tmp")

	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if p.steps[0].name != "tests_pass" {
		t.Errorf("expected name %q, got %q", "tests_pass", p.steps[0].name)
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
