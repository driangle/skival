package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/suite"
)

func TestRunHook_EmptyScript(t *testing.T) {
	hr := runHook(context.Background(), "", "")
	if hr != nil {
		t.Error("expected nil for empty script")
	}
}

func TestRunHook_Success(t *testing.T) {
	hr := runHook(context.Background(), "echo hello", "")
	if hr == nil {
		t.Fatal("expected non-nil result")
	}
	if hr.Err != nil {
		t.Errorf("unexpected error: %v", hr.Err)
	}
	if hr.Stdout != "hello\n" {
		t.Errorf("stdout = %q, want 'hello\\n'", hr.Stdout)
	}
}

func TestRunHook_Failure(t *testing.T) {
	hr := runHook(context.Background(), "exit 1", "")
	if hr == nil || hr.Err == nil {
		t.Error("expected error for failing hook")
	}
}

func TestRunHook_WorkingDir(t *testing.T) {
	dir := t.TempDir()
	hr := runHook(context.Background(), "pwd", dir)
	if hr == nil {
		t.Fatal("expected non-nil result")
	}
	if hr.Err != nil {
		t.Fatalf("unexpected error: %v", hr.Err)
	}
	// pwd output should contain the temp dir
	got := filepath.Clean(hr.Stdout[:len(hr.Stdout)-1]) // trim newline
	want := filepath.Clean(dir)
	// On macOS, /tmp may resolve to /private/tmp
	resolvedGot, _ := filepath.EvalSymlinks(got)
	resolvedWant, _ := filepath.EvalSymlinks(want)
	if resolvedGot != resolvedWant {
		t.Errorf("pwd = %q, want %q", resolvedGot, resolvedWant)
	}
}

func TestExecuteEval_BeforeHookRuns(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "before-ran")

	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Dir:    dir,
			Setup: suite.Setup{
				Before: "touch " + marker,
			},
			Treatments: suite.Treatments{
				Control: suite.Treatment{Name: "ctrl"},
			},
		}},
	}

	sr, err := Execute(context.Background(), s, runner, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sr.Evals[0].Err != nil {
		t.Fatalf("eval error: %v", sr.Evals[0].Err)
	}

	if _, err := os.Stat(marker); err != nil {
		t.Error("before hook did not run")
	}
}

func TestExecuteEval_BeforeHookFailure(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Setup: suite.Setup{
				Before: "exit 1",
			},
			Treatments: suite.Treatments{
				Control: suite.Treatment{Name: "ctrl"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, runner, nil)
	if sr.Evals[0].Err == nil {
		t.Error("expected eval error when before hook fails")
	}
	if len(sr.Evals[0].Treatments) != 0 {
		t.Error("no treatments should run when before hook fails")
	}
}

func TestExecuteEval_ResetBetweenTreatments(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "reset-count")

	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "r1"}, {Text: "r2"}, {Text: "r3"},
		},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Dir:    dir,
			Setup: suite.Setup{
				Reset: "echo x >> " + counter,
			},
			Treatments: suite.Treatments{
				Control:    suite.Treatment{Name: "ctrl"},
				Variations: []suite.Treatment{{Name: "v1"}, {Name: "v2"}},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, runner, nil)
	if sr.Evals[0].Err != nil {
		t.Fatalf("eval error: %v", sr.Evals[0].Err)
	}

	// 3 treatments, reset runs between them = 2 times
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatalf("reset hook did not run: %v", err)
	}
	lines := 0
	for _, b := range data {
		if b == 'x' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("reset ran %d times, want 2", lines)
	}
}

func TestExecuteEval_AfterHookRunsOnError(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "after-ran")

	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Dir:    dir,
			Setup: suite.Setup{
				Before: "exit 1",
				After:  "touch " + marker,
			},
			Treatments: suite.Treatments{
				Control: suite.Treatment{Name: "ctrl"},
			},
		}},
	}

	_, _ = Execute(context.Background(), s, runner, nil)

	if _, err := os.Stat(marker); err != nil {
		t.Error("after hook should run even when before hook fails")
	}
}

func TestExecuteEval_ResetHookFailure(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{
			{Text: "r1"}, {Text: "r2"},
		},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Setup: suite.Setup{
				Reset: "exit 1",
			},
			Treatments: suite.Treatments{
				Control:    suite.Treatment{Name: "ctrl"},
				Variations: []suite.Treatment{{Name: "v1"}},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, runner, nil)
	if sr.Evals[0].Err == nil {
		t.Error("expected eval error when reset hook fails")
	}
	// First treatment should have run, second should not
	if len(sr.Evals[0].Treatments) != 1 {
		t.Errorf("expected 1 treatment (before reset failure), got %d", len(sr.Evals[0].Treatments))
	}
}
