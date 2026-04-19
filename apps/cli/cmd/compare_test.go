package cmd

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/driangle/skival/internal/persist"
	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
)

func saveTestResults(t *testing.T, dir string, sr *result.SuiteResult) string {
	t.Helper()
	saved, err := persist.Save(dir, sr, report.DefaultWeights())
	if err != nil {
		t.Fatalf("saving test results: %v", err)
	}
	return saved
}

func makeSuiteResult(desc string, passRate float64, cost float64, durationMs int64) *result.SuiteResult {
	pass := passRate >= 1.0
	runs := []result.RunResult{
		{Sample: 1, Pass: &pass, CostUSD: cost, DurationMs: durationMs},
	}
	return &result.SuiteResult{
		Description: desc,
		StartedAt:   time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 4, 18, 10, 30, 0, 0, time.UTC),
		Evals: []result.EvalResult{
			{
				EvalID:   "eval-1",
				EvalName: "Test Eval",
				Variants: []result.VariantResult{
					{Name: "control", Runs: runs, Aggregate: result.ComputeAggregate(runs)},
				},
			},
		},
	}
}

func TestCompareCmd_Markdown(t *testing.T) {
	dir := t.TempDir()
	baselineDir := saveTestResults(t, filepath.Join(dir, "baseline"), makeSuiteResult("baseline", 1.0, 0.05, 5000))
	candidateDir := saveTestResults(t, filepath.Join(dir, "candidate"), makeSuiteResult("candidate", 1.0, 0.03, 3000))

	cmd := compareCmd
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{baselineDir, candidateDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	assertContains(t, out, "Comparison Report")
	assertContains(t, out, "control")
	assertContains(t, out, "matched")
}

func TestCompareCmd_JSON(t *testing.T) {
	dir := t.TempDir()
	baselineDir := saveTestResults(t, filepath.Join(dir, "baseline"), makeSuiteResult("baseline", 1.0, 0.05, 5000))
	candidateDir := saveTestResults(t, filepath.Join(dir, "candidate"), makeSuiteResult("candidate", 1.0, 0.03, 3000))

	cmd := compareCmd
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("setting format flag: %v", err)
	}
	defer func() {
		if err := cmd.Flags().Set("format", "markdown"); err != nil {
			t.Fatalf("resetting format flag: %v", err)
		}
	}()

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{baselineDir, candidateDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	assertContains(t, out, `"baseline"`)
	assertContains(t, out, `"eval_id"`)
	assertContains(t, out, `"matched"`)
}

func TestCompareCmd_InvalidBaselineDir(t *testing.T) {
	dir := t.TempDir()
	candidateDir := saveTestResults(t, filepath.Join(dir, "candidate"), makeSuiteResult("candidate", 1.0, 0.05, 5000))

	cmd := compareCmd
	err := cmd.RunE(cmd, []string{"/nonexistent/dir", candidateDir})
	if err == nil {
		t.Fatal("expected error for missing baseline directory")
	}
	assertContains(t, err.Error(), "baseline")
}

func TestCompareCmd_InvalidCandidateDir(t *testing.T) {
	dir := t.TempDir()
	baselineDir := saveTestResults(t, filepath.Join(dir, "baseline"), makeSuiteResult("baseline", 1.0, 0.05, 5000))

	cmd := compareCmd
	err := cmd.RunE(cmd, []string{baselineDir, "/nonexistent/dir"})
	if err == nil {
		t.Fatal("expected error for missing candidate directory")
	}
	assertContains(t, err.Error(), "candidate")
}
