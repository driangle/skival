package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/result"
)

func passRuns(passes ...bool) []result.RunResult {
	runs := make([]result.RunResult, len(passes))
	for i, p := range passes {
		runs[i] = result.RunResult{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(p)}
	}
	return runs
}

func TestWriteMarkdown_RankingCIColumn(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runs: passRuns(true, true, false)},
				{Name: "b", Runs: passRuns(false, false, false)},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "95% CI") {
		t.Error("ranking table missing 95% CI column header")
	}
	// The CI cell is rendered like [lo–hi%].
	if !strings.Contains(out, "–") || !strings.Contains(out, "%]") {
		t.Errorf("ranking table missing rendered CI cell, got:\n%s", out)
	}
}

// Two variants with small, close samples have overlapping Wilson intervals, so
// the report warns the comparison is not significant.
func TestWriteMarkdown_NotSignificantNote(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runs: passRuns(true, true, false)},
				{Name: "b", Runs: passRuns(true, false, false)},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "not significant at this sample size") {
		t.Errorf("expected not-significant note for overlapping intervals, got:\n%s", out)
	}
}

// A large, clearly-separated sample (all-pass vs none-pass at n=12) yields
// disjoint Wilson intervals, so no note is printed.
func TestWriteMarkdown_SignificantNoNote(t *testing.T) {
	allPass := passRuns(true, true, true, true, true, true, true, true, true, true, true, true)
	nonePass := passRuns(false, false, false, false, false, false, false, false, false, false, false, false)
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runs: allPass},
				{Name: "b", Runs: nonePass},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if strings.Contains(out, "not significant at this sample size") {
		t.Errorf("did not expect not-significant note for disjoint intervals, got:\n%s", out)
	}
}
