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
	WriteMarkdown(&buf, sr, DefaultWeights())
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
			Variants: []result.VariantResult{{
				Name: "control",
				Runs: []result.RunResult{
					{Sample: 1, CostUSD: 0.1234, DurationMs: 2500, Pass: boolPtr(true)},
				},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "fizzbuzz") {
		t.Error("missing eval name")
	}
	if !strings.Contains(out, "control") {
		t.Error("missing variant name")
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
			Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "## Rankings") {
		t.Error("missing rankings section")
	}
	if !strings.Contains(out, "#1") {
		t.Error("missing rank #1")
	}
}

func TestWriteMarkdown_NoRankingForSingleVariant(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "only", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if strings.Contains(out, "## Rankings") {
		t.Error("should not show rankings for single variant")
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
	WriteMarkdown(&buf, sr, DefaultWeights())
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
			Variants: []result.VariantResult{{
				Name: "ctrl",
				Runs: []result.RunResult{{Sample: 1, CostUSD: 0.1, DurationMs: 100}},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())

	if strings.Contains(buf.String(), "## Errors") {
		t.Error("should not show errors section when there are no eval errors")
	}
}

func TestWriteMarkdown_RunnerAnnotationMultipleRunners(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Variants: []result.VariantResult{
				{Name: "a", Runner: "claude-code", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runner: "ollama", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "a (claude-code)") {
		t.Error("expected runner annotation on variant a")
	}
	if !strings.Contains(out, "b (ollama)") {
		t.Error("expected runner annotation on variant b")
	}
}

func TestWriteMarkdown_NoRunnerAnnotationSingleRunner(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Variants: []result.VariantResult{
				{Name: "a", Runner: "claude-code", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runner: "claude-code", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if strings.Contains(out, "(claude-code)") {
		t.Error("should not show runner annotation when all variants use the same runner")
	}
}

func TestWriteMarkdown_RankingRunnerColumn(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Runner: "claude-code", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runner: "ollama", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "RUNNER") {
		t.Error("expected RUNNER column header in rankings for multi-runner suite")
	}
	if !strings.Contains(out, "claude-code") {
		t.Error("expected runner name in rankings")
	}
}

func TestWriteMarkdown_ModelAnnotationMultipleModels(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Variants: []result.VariantResult{
				{Name: "a", Runner: "claude-code", Model: "claude-sonnet-4-6", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runner: "claude-code", Model: "claude-opus-4-6", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "claude-sonnet-4-6") {
		t.Error("expected model annotation for variant a")
	}
	if !strings.Contains(out, "claude-opus-4-6") {
		t.Error("expected model annotation for variant b")
	}
}

func TestWriteMarkdown_NoModelAnnotationSingleModel(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Variants: []result.VariantResult{
				{Name: "a", Model: "claude-sonnet-4-6", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Model: "claude-sonnet-4-6", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if strings.Contains(out, "claude-sonnet-4-6") {
		t.Error("should not show model annotation when all variants use the same model")
	}
}

func TestWriteMarkdown_RankingModelColumn(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "a", Model: "claude-sonnet-4-6", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Model: "claude-opus-4-6", Runs: []result.RunResult{{CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "MODEL") {
		t.Error("expected MODEL column header in rankings for multi-model suite")
	}
}

func TestWriteMarkdown_SkippedVariants(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "my-eval",
			Err:      fmt.Errorf("setup.before: hook failed: boom"),
			Skipped: []result.SkippedVariant{
				{Name: "ctrl", Reason: "before hook failed"},
				{Name: "v1", Reason: "before hook failed"},
			},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "## Skipped Variants") {
		t.Error("missing skipped variants section")
	}
	if !strings.Contains(out, "ctrl") {
		t.Error("missing skipped variant name 'ctrl'")
	}
	if !strings.Contains(out, "v1") {
		t.Error("missing skipped variant name 'v1'")
	}
	if !strings.Contains(out, "before hook failed") {
		t.Error("missing skip reason")
	}
}

func TestWriteMarkdown_NoSkippedSectionWhenNoneSkipped(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "good",
			Variants: []result.VariantResult{{
				Name: "ctrl",
				Runs: []result.RunResult{{Sample: 1, CostUSD: 0.1, DurationMs: 100}},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())

	if strings.Contains(buf.String(), "## Skipped") {
		t.Error("should not show skipped section when there are no skipped variants")
	}
}

func TestWriteMarkdown_AggregateRow(t *testing.T) {
	cv := 0.15
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Variants: []result.VariantResult{{
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
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "agg") {
		t.Error("missing aggregate row")
	}
	if !strings.Contains(out, "cost_cv=15.0%") {
		t.Error("missing CV info")
	}
}
