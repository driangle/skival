package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

// renderHTML renders a suite to HTML, failing the test on any error.
func renderHTML(t *testing.T, sr *result.SuiteResult) string {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteHTML(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return buf.String()
}

func wantAll(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output", want)
		}
	}
}

func TestWriteHTML_ValidDocument(t *testing.T) {
	sr := &result.SuiteResult{StartedAt: time.Now(), FinishedAt: time.Now()}
	wantAll(t, renderHTML(t, sr), "<!DOCTYPE html>", "</html>", "Eval Report", "skival eval report", "Run health")
}

func TestWriteHTML_Header(t *testing.T) {
	sr := &result.SuiteResult{
		Description: "My test suite",
		StartedAt:   time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
	}
	wantAll(t, renderHTML(t, sr), "My test suite", "2026-03-19", "300.0s wall")
}

func TestWriteHTML_ResultsTable(t *testing.T) {
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
	wantAll(t, renderHTML(t, sr), "fizzbuzz", "control", "$0.1234", "2.5s", "pass", `data-variant="control"`)
}

func TestWriteHTML_RankingTable(t *testing.T) {
	out := renderHTML(t, twoVariantSuite())
	wantAll(t, out, "Rankings", "#1", "composite", "Verdict", "wins on composite score")
}

// The verdict block must be derivable from the rankings it sits above: the
// winner is rank #1 and the margin is signed against the runner-up.
func TestWriteHTML_VerdictMatchesRankings(t *testing.T) {
	out := renderHTML(t, twoVariantSuite())
	if !strings.Contains(out, "<b>a</b>") {
		t.Error("winner should be variant a")
	}
	// The "+" of the signed margin is emitted as the HTML entity &#43; by
	// html/template's text escaper; browsers render it back to "+".
	if !strings.Contains(out, "&#43;0.") {
		t.Error("missing positive score margin against runner-up")
	}
	if !strings.Contains(out, "× faster") && !strings.Contains(out, "× slower") {
		t.Error("missing speed ratio against runner-up")
	}
}

// Run health counts every sample run, so the headline can never disagree with
// the strip of per-sample cells beneath it.
func TestWriteHTML_RunHealth(t *testing.T) {
	out := renderHTML(t, twoVariantSuite())
	wantAll(t, out, "1/2", "samples passed", `class="health-cell`, `class="health-cell fail"`, "$3.0000")
}

func TestWriteHTML_NoRankingForSingleVariant(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "only", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100}}},
			},
		}},
	}
	out := renderHTML(t, sr)
	if strings.Contains(out, "Rankings") {
		t.Error("should not show rankings for single variant")
	}
	if !strings.Contains(out, "nothing to compare") {
		t.Error("should explain why there is no verdict")
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
	wantAll(t, renderHTML(t, sr), "Errors", "broken eval", "hook failed")
}

func TestWriteHTML_SkippedVariants(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "my-eval",
			Err:      fmt.Errorf("hook failed"),
			Skipped: []result.SkippedVariant{
				{Name: "ctrl", Reason: "before hook failed"},
			},
		}},
	}
	wantAll(t, renderHTML(t, sr), "my-eval", "ctrl", "before hook failed")
}

func TestWriteHTML_StatusClasses(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "test",
			Variants: []result.VariantResult{{
				Name: "ctrl",
				Runs: []result.RunResult{
					{Sample: 1, CostUSD: 0.1, DurationMs: 100, Pass: boolPtr(true)},
					{Sample: 2, CostUSD: 0.2, DurationMs: 200, Pass: boolPtr(false)},
				},
			}},
		}},
	}
	wantAll(t, renderHTML(t, sr), `class="status pass"`, `class="status fail"`)
}

func TestWriteHTML_InlinesEmbeddedAssets(t *testing.T) {
	sr := &result.SuiteResult{StartedAt: time.Now(), FinishedAt: time.Now()}
	out := renderHTML(t, sr)

	if !strings.Contains(out, ".status.pass { color: var(--pass)") {
		t.Error("embedded CSS not inlined into <style>")
	}
	if !strings.Contains(out, "window.sortTable = function") {
		t.Error("embedded JS not inlined into <script>")
	}
	if strings.Contains(out, "&lt;") || strings.Contains(out, "\\u003c") {
		t.Error("inlined assets appear HTML/JS-escaped; template.CSS/template.JS not honored")
	}
}

func TestWriteHTML_Interactions(t *testing.T) {
	out := renderHTML(t, twoVariantSuite())
	wantAll(t, out, "sortTable(this)", "toggleEval(this)", "filterVariant(this)", "toggleTheme()")
}

func TestWriteHTML_AggregateRow(t *testing.T) {
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
				},
				Aggregate: &result.Aggregate{
					MedianCostUSD: 1.5, MinCostUSD: 1.0, MaxCostUSD: 2.0,
					MedianDurationMs: 150, MinDurationMs: 100, MaxDurationMs: 200,
					CostCV: &cv,
				},
			}},
		}},
	}
	out := renderHTML(t, sr)
	wantAll(t, out, `class="agg"`, "cost cv 15%", "median")

	// The spread bar spans min→max within the eval's slowest run.
	if !strings.Contains(out, "left:50.0%;width:50.0%") {
		t.Error("missing min-max spread band on aggregate row")
	}
}

func TestWriteHTML_MultiRunnerAttribution(t *testing.T) {
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
	wantAll(t, renderHTML(t, sr), "claude-code", "ollama", `class="rank-model"`)
}

func TestWriteHTML_JudgeVerdicts(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "explain",
			Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
			},
			Comparison: &result.Comparison{
				Model: "claude-haiku-4-5",
				Scores: []result.ComparativeScore{
					{Variant: "a", Rating: 4, Score: 0.8, Reason: "Covers the trade-offs. Also concise."},
				},
			},
		}},
	}
	out := renderHTML(t, sr)
	wantAll(t, out, "Judge verdict", "claude-haiku-4-5", "4/5", "0.80", "Covers the trade-offs.", `<i class="on">`)
}

// A sample run that errors must surface its reason, not just a red status pill,
// so a reader can tell why it failed (e.g. an unknown-runner misconfiguration).
func TestWriteHTML_RunErrorDetail(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "matrix",
			Variants: []result.VariantResult{
				{Name: "ok", Runs: []result.RunResult{{Sample: 1, CostUSD: 0.1, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "codex", Runs: []result.RunResult{{Sample: 1, Err: fmt.Errorf(`creating runner "codex": unknown runner`)}}},
			},
		}},
	}
	out := renderHTML(t, sr)
	wantAll(t, out, `class="status error"`, `class="err-row"`, `class="err-detail"`,
		"toggleRow(this)", `colspan="6"`, "unknown runner")
}

// An IsError run with no error value still gets an expandable marker rather than
// a bare pill.
func TestWriteHTML_RunErrorWithoutMessage(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "e",
			Variants: []result.VariantResult{{Name: "a", Runs: []result.RunResult{{Sample: 2, IsError: true}}}},
		}},
	}
	wantAll(t, renderHTML(t, sr), `class="err-detail"`, "run errored without a message")
}

// A completed (non-errored) run must not become an expandable error row.
func TestWriteHTML_NoErrorRowForPassingRun(t *testing.T) {
	out := renderHTML(t, twoVariantSuite())
	if strings.Contains(out, `class="err-detail"`) {
		t.Error("passing/failing runs should not render error-detail rows")
	}
}

func TestWriteHTML_ComparisonSkipped(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName:   "explain",
			Variants:   []result.VariantResult{{Name: "a", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100}}}},
			Comparison: &result.Comparison{Skipped: "fewer than two passing variants"},
		}},
	}
	wantAll(t, renderHTML(t, sr), "Comparative judging skipped", "fewer than two passing variants")
}

// twoVariantSuite is the smallest suite that exercises the verdict, rankings,
// and health blocks: one passing variant and one failing variant.
func twoVariantSuite() *result.SuiteResult {
	return &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalName: "eval-one",
			Variants: []result.VariantResult{
				{Name: "a", Runs: []result.RunResult{{Sample: 1, CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{Sample: 1, CostUSD: 2.0, DurationMs: 200, Pass: boolPtr(false)}}},
			},
		}},
	}
}
