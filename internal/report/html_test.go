package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

func TestWriteHTML_ValidDocument(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"<!DOCTYPE html>", "<table>", "</html>", "Eval Report"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output", want)
		}
	}
}

func TestWriteHTML_Header(t *testing.T) {
	sr := &result.SuiteResult{
		Description: "My test suite",
		StartedAt:   time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "My test suite") {
		t.Error("missing description")
	}
	if !strings.Contains(out, "2026-03-19") {
		t.Error("missing date")
	}
}

func TestWriteHTML_ResultsTable(t *testing.T) {
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
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"fizzbuzz", "control", "$0.1234", "2.5s", "pass"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in results table", want)
		}
	}
}

func TestWriteHTML_RankingTable(t *testing.T) {
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
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Rankings") {
		t.Error("missing rankings section")
	}
	if !strings.Contains(out, "#1") {
		t.Error("missing rank #1")
	}
}

func TestWriteHTML_NoRankingForSingleTreatment(t *testing.T) {
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
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(buf.String(), "Rankings") {
		t.Error("should not show rankings for single treatment")
	}
}

func TestWriteHTML_Errors(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "broken",
			EvalName: "broken eval",
			Err:      fmt.Errorf("setup.before: hook failed"),
		}},
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Errors") {
		t.Error("missing errors section")
	}
	if !strings.Contains(out, "broken eval") {
		t.Error("missing eval name in errors")
	}
	if !strings.Contains(out, "hook failed") {
		t.Error("missing error message")
	}
}

func TestWriteHTML_SkippedTreatments(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "my-eval",
			Err:      fmt.Errorf("hook failed"),
			Skipped: []result.SkippedTreatment{
				{Name: "ctrl", Reason: "before hook failed"},
			},
		}},
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Skipped Treatments") {
		t.Error("missing skipped treatments section")
	}
	if !strings.Contains(out, "ctrl") {
		t.Error("missing skipped treatment name")
	}
	if !strings.Contains(out, "before hook failed") {
		t.Error("missing skip reason")
	}
}

func TestWriteHTML_StatusColors(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Treatments: []result.TreatmentResult{{
				Name: "ctrl",
				Runs: []result.RunResult{
					{Sample: 1, CostUSD: 0.1, DurationMs: 100, Pass: boolPtr(true)},
					{Sample: 2, CostUSD: 0.2, DurationMs: 200, Pass: boolPtr(false)},
				},
			}},
		}},
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, `class="status-pass"`) {
		t.Error("missing pass status class")
	}
	if !strings.Contains(out, `class="status-fail"`) {
		t.Error("missing fail status class")
	}
}

func TestWriteHTML_SortableHeaders(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "onclick=") {
		t.Error("missing sortable onclick handlers on table headers")
	}
	if !strings.Contains(out, "sortTable") {
		t.Error("missing sortTable function")
	}
}

func TestWriteHTML_AggregateRow(t *testing.T) {
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
				},
				Aggregate: &result.Aggregate{
					MedianCostUSD: 1.5, MinCostUSD: 1.0, MaxCostUSD: 2.0,
					MedianDurationMs: 150, MinDurationMs: 100, MaxDurationMs: 200,
					CostCV: &cv,
				},
			}},
		}},
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, `class="agg"`) {
		t.Error("missing aggregate row class")
	}
	if !strings.Contains(out, "cost_cv=15.0%") {
		t.Error("missing CV info")
	}
}

func TestWriteHTML_MultiRunnerColumns(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{Name: "a", Runner: "claude-code", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runner: "ollama", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, ">Runner<") {
		t.Error("missing Runner column header in rankings")
	}
}
