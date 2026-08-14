package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

// TestSamplesFlagDefaultIsUnsetSentinel guards against regression where the
// --samples flag default was 1, which silently overrode suite.yaml's
// `defaults.samples` because the executor treats any positive value as a CLI
// override. The default must be 0 so the YAML value is honored.
func TestSamplesFlagDefaultIsUnsetSentinel(t *testing.T) {
	f := runCmd.Flags().Lookup("samples")
	if f == nil {
		t.Fatal("expected --samples flag to be registered")
	}
	if f.DefValue != "0" {
		t.Errorf("expected --samples default to be 0 (unset sentinel), got %q", f.DefValue)
	}
}

func TestDefaultRegistry(t *testing.T) {
	reg := defaultRegistry()

	for _, name := range []string{"claude-code", "ollama"} {
		runner, err := reg.Create(name, nil)
		if err != nil {
			t.Errorf("expected %q to be registered, got error: %v", name, err)
		}
		if runner == nil {
			t.Errorf("expected non-nil runner for %q", name)
		}
	}

	_, err := reg.Create("nonexistent", nil)
	if err == nil {
		t.Error("expected error for unregistered runner")
	}
}

func TestRankingWeights_DefaultsWhenNoCompare(t *testing.T) {
	s := &suite.Suite{}
	w := rankingWeights(s, nil)
	def := report.DefaultWeights()
	if w != def {
		t.Errorf("expected default weights, got %+v", w)
	}
	if w.Quality != 0 {
		t.Errorf("quality weight should be 0 without comparison, got %g", w.Quality)
	}
}

func TestRankingWeights_CarvesQualityWhenCompareActive(t *testing.T) {
	s := &suite.Suite{
		Compare: &suite.Compare{Criteria: []string{"clarity"}},
		Evals:   []suite.Eval{{ID: "e1"}},
	}
	w := rankingWeights(s, nil)
	if w.Quality != suite.DefaultCompareWeight {
		t.Errorf("quality weight = %g, want %g", w.Quality, suite.DefaultCompareWeight)
	}
	sum := w.Correctness + w.Cost + w.Duration + w.Quality
	if sum < 0.999999 || sum > 1.000001 {
		t.Errorf("weights should sum to 1.0, got %g", sum)
	}
}

func TestRankingWeights_ExplicitRankingIncludesQuality(t *testing.T) {
	s := &suite.Suite{
		Ranking: &suite.Ranking{Weights: suite.RankingWeights{Correctness: 0.5, Cost: 0.2, Duration: 0.1, Quality: 0.2}},
	}
	w := rankingWeights(s, nil)
	if w.Quality != 0.2 {
		t.Errorf("explicit quality weight not honored, got %g", w.Quality)
	}
}

func TestRankingWeights_NoCompareOverrideDropsQuality(t *testing.T) {
	s := &suite.Suite{
		Compare: &suite.Compare{Criteria: []string{"clarity"}},
		Evals:   []suite.Eval{{ID: "e1"}},
	}
	off := false
	w := rankingWeights(s, &off)
	if w.Quality != 0 {
		t.Errorf("--no-compare should drop quality weight, got %g", w.Quality)
	}
}

func TestKeepWorkdirsFlagDefault(t *testing.T) {
	f := runCmd.Flags().Lookup("keep-workdirs")
	if f == nil {
		t.Fatal("expected --keep-workdirs flag to be registered")
	}
	if f.DefValue != "failed" {
		t.Errorf("expected --keep-workdirs default to be \"failed\", got %q", f.DefValue)
	}
}

func TestRunCmd_InvalidKeepWorkdirsErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "suite.yaml")
	content := `
version: 1
description: "test suite"
defaults:
  runner: claude-code
evals:
  - id: eval-1
    name: "Test Eval"
    prompt: "do something"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    variants:
      - name: "baseline"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runCmd.Flags().Set("keep-workdirs", "bogus"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runCmd.Flags().Set("keep-workdirs", "failed") })

	err := runCmd.RunE(runCmd, []string{path})
	if err == nil {
		t.Fatal("expected error for invalid --keep-workdirs value")
	}
	if !strings.Contains(err.Error(), "keep-workdirs") {
		t.Errorf("expected error to mention keep-workdirs, got: %v", err)
	}
}

func TestCompareFlagsRegistered(t *testing.T) {
	if runCmd.Flags().Lookup("compare") == nil {
		t.Error("--compare flag not registered")
	}
	if runCmd.Flags().Lookup("no-compare") == nil {
		t.Error("--no-compare flag not registered")
	}
}

func TestCostFlagsRegistered(t *testing.T) {
	if runCmd.Flags().Lookup("dry-run") == nil {
		t.Error("--dry-run flag not registered")
	}
	if f := runCmd.Flags().Lookup("max-cost"); f == nil {
		t.Error("--max-cost flag not registered")
	} else if f.DefValue != "0" {
		t.Errorf("expected --max-cost default 0, got %q", f.DefValue)
	}
}

const dryRunSuite = `
version: 1
description: "test suite"
defaults:
  runner: claude-code
evals:
  - id: eval-1
    name: "First Eval"
    prompt: "do something"
    model: "claude-sonnet-4-6"
    samples: 2
    verify:
      - type: agent_exits_ok
    variants:
      - name: "baseline"
      - name: "treatment"
        model: "claude-opus-4-6"
`

// captureStdout redirects os.Stdout for the duration of fn and returns what was
// written. The run command prints the dry-run matrix to os.Stdout directly.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func writeSuite(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "suite.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunCmd_DryRunPrintsMatrixWithoutExecuting(t *testing.T) {
	path := writeSuite(t, dryRunSuite)

	if err := runCmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runCmd.Flags().Set("dry-run", "false") })

	var runErr error
	out := captureStdout(t, func() { runErr = runCmd.RunE(runCmd, []string{path}) })
	if runErr != nil {
		t.Fatalf("dry-run should not error, got: %v", runErr)
	}

	for _, want := range []string{"First Eval", "baseline", "treatment", "claude-opus-4-6", "dry run"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected dry-run output to contain %q, got:\n%s", want, out)
		}
	}
	// 2 variants × 2 samples = 4 total samples, nothing executed.
	if !strings.Contains(out, "4 total samples") {
		t.Errorf("expected 4 total samples in output, got:\n%s", out)
	}
}

func TestAbortError(t *testing.T) {
	if err := abortError(&result.SuiteResult{}); err != nil {
		t.Errorf("expected nil error for a completed run, got: %v", err)
	}

	sr := &result.SuiteResult{Abort: &result.Abort{Reason: "cost cap exceeded", SpentUSD: 0.15, CapUSD: 0.10}}
	err := abortError(sr)
	if err == nil {
		t.Fatal("expected non-nil error for an aborted run")
	}
	if !strings.Contains(err.Error(), "max-cost") {
		t.Errorf("expected abort error to mention max-cost, got: %v", err)
	}
}

func TestRunCmd_NegativeMaxCostErrors(t *testing.T) {
	path := writeSuite(t, dryRunSuite)

	if err := runCmd.Flags().Set("max-cost", "-1"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runCmd.Flags().Set("max-cost", "0") })

	err := runCmd.RunE(runCmd, []string{path})
	if err == nil {
		t.Fatal("expected error for negative --max-cost")
	}
	if !strings.Contains(err.Error(), "max-cost") {
		t.Errorf("expected error to mention max-cost, got: %v", err)
	}
}
