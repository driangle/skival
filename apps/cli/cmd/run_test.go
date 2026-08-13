package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/report"
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
