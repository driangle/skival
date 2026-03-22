package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

func TestWriteMarkdown_Header(t *testing.T) {
	sr := &result.SuiteResult{
		Description: "Test suite",
		StartedAt:   time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)
	out := buf.String()

	if !strings.Contains(out, "# Eval Report") {
		t.Error("missing report header")
	}
	if !strings.Contains(out, "Test suite") {
		t.Error("missing description")
	}
	if !strings.Contains(out, "2026-03-19") {
		t.Error("missing date")
	}
}

func TestWriteMarkdown_ResultsTable(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "fizzbuzz",
			Treatments: []result.TreatmentResult{{
				Name: "control",
				Runs: []result.RunResult{
					{Sample: 1, CostUSD: 0.1234, DurationMs: 2500, Pass: boolPtr(true)},
				},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)
	out := buf.String()

	if !strings.Contains(out, "fizzbuzz") {
		t.Error("missing eval name")
	}
	if !strings.Contains(out, "control") {
		t.Error("missing treatment name")
	}
	if !strings.Contains(out, "$0.1234") {
		t.Error("missing formatted cost")
	}
	if !strings.Contains(out, "2.5s") {
		t.Error("missing formatted duration")
	}
	if !strings.Contains(out, "pass") {
		t.Error("missing pass status")
	}
}

func TestWriteMarkdown_RankingTable(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)
	out := buf.String()

	if !strings.Contains(out, "## Rankings") {
		t.Error("missing rankings section")
	}
	if !strings.Contains(out, "#1") {
		t.Error("missing rank #1")
	}
}

func TestWriteMarkdown_NoRankingForSingleTreatment(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{Name: "only", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)
	out := buf.String()

	if strings.Contains(out, "## Rankings") {
		t.Error("should not show rankings for single treatment")
	}
}

func TestWriteMarkdown_EvalError(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "broken",
			EvalName: "broken eval",
			Err:      fmt.Errorf("setup.before: hook failed: no such directory"),
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)
	out := buf.String()

	if !strings.Contains(out, "broken eval") {
		t.Error("missing eval name in results table")
	}
	if !strings.Contains(out, "ERROR") {
		t.Error("missing ERROR status in results table")
	}
	if !strings.Contains(out, "## Errors") {
		t.Error("missing errors section")
	}
	if !strings.Contains(out, "no such directory") {
		t.Error("missing error message in errors section")
	}
}

func TestWriteMarkdown_NoErrorsSectionWhenAllPass(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "good",
			Treatments: []result.TreatmentResult{{
				Name: "ctrl",
				Runs: []result.RunResult{{Sample: 1, CostUSD: 0.1, DurationMs: 100}},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)

	if strings.Contains(buf.String(), "## Errors") {
		t.Error("should not show errors section when there are no eval errors")
	}
}

func TestWriteMarkdown_AggregateRow(t *testing.T) {
	cv := 0.15
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Treatments: []result.TreatmentResult{{
				Name: "ctrl",
				Runs: []result.RunResult{
					{Sample: 1, CostUSD: 1.0, DurationMs: 100},
					{Sample: 2, CostUSD: 2.0, DurationMs: 200},
					{Sample: 3, CostUSD: 3.0, DurationMs: 300},
				},
				Aggregate: &result.Aggregate{
					MedianCostUSD: 2.0, MinCostUSD: 1.0, MaxCostUSD: 3.0,
					MedianDurationMs: 200, MinDurationMs: 100, MaxDurationMs: 300,
					CostCV: &cv,
				},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr)
	out := buf.String()

	if !strings.Contains(out, "agg") {
		t.Error("missing aggregate row")
	}
	if !strings.Contains(out, "cost_cv=15.0%") {
		t.Error("missing CV info")
	}
}
