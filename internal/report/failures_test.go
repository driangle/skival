package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

func suiteWithFailedSample() *result.SuiteResult {
	return &result.SuiteResult{
		StartedAt:  time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
		Evals: []result.EvalResult{{
			EvalID:   "eval1",
			EvalName: "fizzbuzz",
			Variants: []result.VariantResult{{
				Name: "control",
				Runs: []result.RunResult{{
					Sample: 2,
					Pass:   boolPtr(false),
					Steps: []result.StepResult{
						{Name: "build", Type: "check", Pass: true},
						{Name: "tests", Type: "check", Pass: false,
							Reason: "check: command failed: FAIL expected 3 got 4"},
					},
				}},
			}},
		}},
	}
}

func TestWriteMarkdown_FailuresSection(t *testing.T) {
	var buf bytes.Buffer
	WriteMarkdown(&buf, suiteWithFailedSample(), DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "## Failures") {
		t.Fatalf("missing Failures section:\n%s", out)
	}
	if !strings.Contains(out, "fizzbuzz") || !strings.Contains(out, "sample 2") {
		t.Errorf("failure line missing eval/sample:\n%s", out)
	}
	if !strings.Contains(out, "tests") {
		t.Errorf("failure line should name the failing step:\n%s", out)
	}
	if !strings.Contains(out, "FAIL expected 3 got 4") {
		t.Errorf("failure line should include the reason:\n%s", out)
	}
}

func TestWriteMarkdown_NoFailuresSectionWhenAllPass(t *testing.T) {
	sr := suiteWithFailedSample()
	sr.Evals[0].Variants[0].Runs[0].Pass = boolPtr(true)

	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	if strings.Contains(buf.String(), "## Failures") {
		t.Error("should not render Failures section when nothing failed")
	}
}
