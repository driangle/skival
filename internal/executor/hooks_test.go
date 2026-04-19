package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

func TestRunHook_EmptyScript(t *testing.T) {
	he := runHook(context.Background(), "", "")
	if he != nil {
		t.Error("expected nil for empty script")
	}
}

func TestRunHook_Success(t *testing.T) {
	he := runHook(context.Background(), "echo hello", "")
	if he != nil {
		t.Errorf("unexpected error: %v", he)
	}
}

func TestRunHook_Failure(t *testing.T) {
	he := runHook(context.Background(), "exit 1", "")
	if he == nil {
		t.Error("expected error for failing hook")
	}
}

func TestRunHook_FailureCapturesStderr(t *testing.T) {
	he := runHook(context.Background(), "echo 'boom' >&2; exit 1", "")
	if he == nil {
		t.Fatal("expected error for failing hook")
	}
	if he.Stderr == "" {
		t.Error("expected stderr to be captured")
	}
	errMsg := he.Error()
	if !strings.Contains(errMsg, "boom") {
		t.Errorf("error message should contain stderr content, got: %s", errMsg)
	}
}

func TestRunHook_FailureCapturesStdout(t *testing.T) {
	he := runHook(context.Background(), "echo 'debug info'; exit 1", "")
	if he == nil {
		t.Fatal("expected error for failing hook")
	}
	if he.Stdout == "" {
		t.Error("expected stdout to be captured")
	}
	errMsg := he.Error()
	if !strings.Contains(errMsg, "debug info") {
		t.Errorf("error message should contain stdout content, got: %s", errMsg)
	}
}

func TestRunHook_WorkingDir(t *testing.T) {
	he := runHook(context.Background(), "pwd", t.TempDir())
	if he != nil {
		t.Fatalf("unexpected error: %v", he)
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
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
			},
		}},
	}

	sr, err := Execute(context.Background(), s, fakeRegistry(runner), nil)
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
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, fakeRegistry(runner), nil)
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
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
				{Name: "v1", Runner: "claude-code"},
				{Name: "v2", Runner: "claude-code"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, fakeRegistry(runner), nil)
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
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
			},
		}},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

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
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
				{Name: "v1", Runner: "claude-code"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, fakeRegistry(runner), nil)
	if sr.Evals[0].Err == nil {
		t.Error("expected eval error when reset hook fails")
	}
	// First treatment should have run, second should not
	if len(sr.Evals[0].Treatments) != 1 {
		t.Errorf("expected 1 treatment (before reset failure), got %d", len(sr.Evals[0].Treatments))
	}
}

func TestExecuteEval_BeforeHookFailure_SkipsAllTreatments(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Setup: suite.Setup{
				Before: "echo 'setup broke' >&2; exit 1",
			},
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
				{Name: "v1", Runner: "claude-code"},
				{Name: "v2", Runner: "claude-code"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, fakeRegistry(runner), nil)
	eval := sr.Evals[0]

	if eval.Err == nil {
		t.Fatal("expected eval error")
	}
	// Error should contain the stderr output.
	if !strings.Contains(eval.Err.Error(), "setup broke") {
		t.Errorf("error should contain hook stderr, got: %v", eval.Err)
	}
	// All 3 treatments should be skipped.
	if len(eval.Skipped) != 3 {
		t.Fatalf("expected 3 skipped treatments, got %d", len(eval.Skipped))
	}
	for _, s := range eval.Skipped {
		if s.Reason != "before hook failed" {
			t.Errorf("expected reason 'before hook failed', got %q", s.Reason)
		}
	}
	wantNames := []string{"ctrl", "v1", "v2"}
	for i, want := range wantNames {
		if eval.Skipped[i].Name != want {
			t.Errorf("skipped[%d].Name = %q, want %q", i, eval.Skipped[i].Name, want)
		}
	}
}

func TestExecuteEval_ResetHookFailure_SkipsRemainingTreatments(t *testing.T) {
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
			Setup: suite.Setup{
				Reset: "echo 'reset broke' >&2; exit 1",
			},
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
				{Name: "v1", Runner: "claude-code"},
				{Name: "v2", Runner: "claude-code"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, fakeRegistry(runner), nil)
	eval := sr.Evals[0]

	if eval.Err == nil {
		t.Fatal("expected eval error")
	}
	errMsg := eval.Err.Error()
	if !strings.Contains(errMsg, "reset broke") {
		t.Errorf("error should contain hook stderr, got: %v", eval.Err)
	}
	// Error should identify the treatments involved.
	if !strings.Contains(errMsg, "ctrl") || !strings.Contains(errMsg, "v1") {
		t.Errorf("error should name affected treatments, got: %s", errMsg)
	}
	// ctrl ran, then reset failed before v1 — v1 and v2 should be skipped.
	if len(eval.Treatments) != 1 {
		t.Errorf("expected 1 completed treatment, got %d", len(eval.Treatments))
	}
	if len(eval.Skipped) != 2 {
		t.Fatalf("expected 2 skipped treatments, got %d", len(eval.Skipped))
	}
	if eval.Skipped[0].Name != "v1" {
		t.Errorf("skipped[0].Name = %q, want v1", eval.Skipped[0].Name)
	}
	if eval.Skipped[1].Name != "v2" {
		t.Errorf("skipped[1].Name = %q, want v2", eval.Skipped[1].Name)
	}
	// Skip reason should mention which treatment completed before the failure.
	if !strings.Contains(eval.Skipped[0].Reason, "ctrl") {
		t.Errorf("skip reason should reference completed treatment, got: %s", eval.Skipped[0].Reason)
	}
}

func TestExecuteEval_BeforeHookFailure_ErrorIncludesOutput(t *testing.T) {
	runner := &fakeRunner{
		results: []*agentrunner.Result{{Text: "ok"}},
	}

	s := &suite.Suite{
		Evals: []suite.Eval{{
			ID:     "e1",
			Name:   "test",
			Prompt: "do it",
			Setup: suite.Setup{
				Before: "echo 'debug log'; echo 'fatal error' >&2; exit 1",
			},
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
			},
		}},
	}

	sr, _ := Execute(context.Background(), s, fakeRegistry(runner), nil)
	eval := sr.Evals[0]

	errMsg := eval.Err.Error()
	if !strings.Contains(errMsg, "fatal error") {
		t.Errorf("error should contain stderr, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "debug log") {
		t.Errorf("error should contain stdout, got: %s", errMsg)
	}
}

func TestExecuteEval_AfterHookRunsOnResetFailure(t *testing.T) {
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
				Reset: "exit 1",
				After: "touch " + marker,
			},
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
				{Name: "v1", Runner: "claude-code"},
			},
		}},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), nil)

	if _, err := os.Stat(marker); err != nil {
		t.Error("after hook should run even when reset hook fails")
	}
}

func TestHookError_IncludesStderrAndStdout(t *testing.T) {
	he := &HookError{
		Message: "hook failed: something",
		Stderr:  "error output\n",
		Stdout:  "normal output\n",
	}
	msg := he.Error()
	if !strings.Contains(msg, "hook failed: something") {
		t.Error("missing message")
	}
	if !strings.Contains(msg, "stderr: error output") {
		t.Error("missing stderr")
	}
	if !strings.Contains(msg, "stdout: normal output") {
		t.Error("missing stdout")
	}
}

func TestHookError_OmitsEmptyOutput(t *testing.T) {
	he := &HookError{
		Message: "hook failed: exit status 1",
	}
	msg := he.Error()
	if strings.Contains(msg, "stderr:") {
		t.Error("should not include stderr label when empty")
	}
	if strings.Contains(msg, "stdout:") {
		t.Error("should not include stdout label when empty")
	}
}

func TestExecuteEval_BeforeHookFailure_SkippedTreatmentsInProgress(t *testing.T) {
	// Verify that skipped treatments are reported via progress output.
	var buf strings.Builder
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
			Variants: []suite.Treatment{
				{Name: "ctrl", Runner: "claude-code"},
				{Name: "v1", Runner: "claude-code"},
			},
		}},
	}

	_, _ = Execute(context.Background(), s, fakeRegistry(runner), &Options{Progress: &buf})

	out := buf.String()
	if !strings.Contains(out, "Skipping 2 remaining treatments") {
		t.Errorf("expected skip message in progress output, got: %s", out)
	}
}

// Verify that SkippedTreatment fields are correctly populated.
func TestSkippedTreatment_Fields(t *testing.T) {
	s := result.SkippedTreatment{
		Name:   "my-treatment",
		Reason: "before hook failed",
	}
	if s.Name != "my-treatment" {
		t.Errorf("Name = %q, want my-treatment", s.Name)
	}
	if s.Reason != "before hook failed" {
		t.Errorf("Reason = %q, want 'before hook failed'", s.Reason)
	}
}
