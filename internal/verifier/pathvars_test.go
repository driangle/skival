package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/driangle/skival/internal/suite"
)

func TestExpandPathVars_SubstitutesSuiteAndWork(t *testing.T) {
	dirs := stepDirs{work: "/work", suite: "/suite"}
	got := expandPathVars("${SKIVAL_SUITE_DIR}/grader.sh --out ${SKIVAL_WORK_DIR}/log", dirs)
	want := "/suite/grader.sh --out /work/log"
	if got != want {
		t.Errorf("expandPathVars = %q, want %q", got, want)
	}
}

func TestExpandPathVars_EmptyStaysEmpty(t *testing.T) {
	if got := expandPathVars("", stepDirs{work: "/w", suite: "/s"}); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExpandPathVars_FallsBackToEnv(t *testing.T) {
	t.Setenv("SKIVAL_PATHVARS_TEST", "envval")
	if got := expandPathVars("${SKIVAL_PATHVARS_TEST}", stepDirs{}); got != "envval" {
		t.Errorf("expected env fallback, got %q", got)
	}
}

// TestPipeline_CheckUsesSuiteDirGrader is the integrity test: a grader living in
// the suite dir (NOT copied into the isolated working dir) is referenced via
// ${SKIVAL_SUITE_DIR} and runs successfully against a work dir that lacks it.
func TestPipeline_CheckUsesSuiteDirGrader(t *testing.T) {
	suiteDir := t.TempDir()
	workDir := t.TempDir()

	// Grader lives beside suite.yaml. It runs with cwd == workDir and asserts
	// the agent produced "marker" there, so a pass proves both substitution and
	// that the grader executed against the working dir.
	grader := filepath.Join(suiteDir, "grader.sh")
	if err := os.WriteFile(grader, []byte("#!/bin/sh\ntest -f marker\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "marker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := BuildPipeline([]suite.VerifyStep{
		{Type: "check", Run: "${SKIVAL_SUITE_DIR}/grader.sh"},
	}, workDir, suiteDir)
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}

	r := p.Run(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Fatalf("expected pass, got steps: %+v", r.Steps)
	}

	// Integrity: the grader must not be present in (nor readable from) the workdir.
	if _, err := os.Stat(filepath.Join(workDir, "grader.sh")); !os.IsNotExist(err) {
		t.Errorf("grader.sh must not exist in workDir, stat err = %v", err)
	}
}

func TestPipeline_CommandUsesWorkDirVar(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "out.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitZero := 0
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "command", Run: "cat ${SKIVAL_WORK_DIR}/out.txt", Exits: &exitZero, StdoutContains: "hello"},
	}, workDir, "/unused-suite-dir")
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}

	r := p.Run(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Fatalf("expected pass, got steps: %+v", r.Steps)
	}
}

// TestPipeline_FileContainsUsesSuiteDirVar verifies a file_contains path using
// ${SKIVAL_SUITE_DIR} resolves against the suite dir (a golden file living next
// to suite.yaml, absent from the working dir).
func TestPipeline_FileContainsUsesSuiteDirVar(t *testing.T) {
	suiteDir := t.TempDir()
	workDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(suiteDir, "golden.txt"), []byte("expected content"), 0o644); err != nil {
		t.Fatal(err)
	}

	trueVal := true
	p := BuildPipeline([]suite.VerifyStep{
		{Type: "file_contains", Path: "${SKIVAL_SUITE_DIR}/golden.txt", Exists: &trueVal, Contains: "expected"},
	}, workDir, suiteDir)
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}

	r := p.Run(context.Background(), VerifyInput{})
	if !r.Pass {
		t.Fatalf("expected pass, got steps: %+v", r.Steps)
	}

	// Integrity: the golden file lives in the suite dir, not the working dir.
	if _, err := os.Stat(filepath.Join(workDir, "golden.txt")); !os.IsNotExist(err) {
		t.Errorf("golden.txt must not exist in workDir, stat err = %v", err)
	}
}
