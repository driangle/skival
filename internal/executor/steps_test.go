package executor

import (
	"context"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/suite"
)

// TestRunSteps_PopulatedOnFailingCheck verifies that a failing verification step
// is preserved on the run's Steps, carrying its name, type, pass state, and the
// reason explaining the failure.
func TestRunSteps_PopulatedOnFailingCheck(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "done"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Verify = []suite.VerifyStep{
		{Type: "check", Name: "always-fails", Run: "exit 3"},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if run.Pass == nil || *run.Pass {
		t.Fatalf("expected run to fail, got Pass=%v", run.Pass)
	}
	if len(run.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d: %+v", len(run.Steps), run.Steps)
	}

	step := run.Steps[0]
	if step.Name != "always-fails" {
		t.Errorf("step name = %q, want always-fails", step.Name)
	}
	if step.Type != "check" {
		t.Errorf("step type = %q, want check", step.Type)
	}
	if step.Pass {
		t.Error("failing step should have Pass=false")
	}
	if step.Reason == "" {
		t.Error("failing step should record a reason")
	}
	if step.ExitCode == nil || *step.ExitCode != 3 {
		t.Errorf("step exit code = %v, want 3", step.ExitCode)
	}
}

// TestRunSteps_PopulatedOnPassingCheck verifies passing steps are also recorded.
func TestRunSteps_PopulatedOnPassingCheck(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "done"}},
	}

	s := newMinimalSuite()
	s.Evals[0].Verify = []suite.VerifyStep{
		{Type: "check", Name: "always-passes", Run: "true"},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run := sr.Evals[0].Variants[0].Runs[0]
	if run.Pass == nil || !*run.Pass {
		t.Fatalf("expected run to pass, got Pass=%v", run.Pass)
	}
	if len(run.Steps) != 1 || !run.Steps[0].Pass {
		t.Fatalf("expected 1 passing step, got %+v", run.Steps)
	}
	if run.Steps[0].Type != "check" {
		t.Errorf("step type = %q, want check", run.Steps[0].Type)
	}
}
