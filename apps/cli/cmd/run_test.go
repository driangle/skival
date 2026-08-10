package cmd

import (
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

func TestCompareFlagsRegistered(t *testing.T) {
	if runCmd.Flags().Lookup("compare") == nil {
		t.Error("--compare flag not registered")
	}
	if runCmd.Flags().Lookup("no-compare") == nil {
		t.Error("--no-compare flag not registered")
	}
}
