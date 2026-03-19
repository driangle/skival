package executor

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/result"
)

func TestPrintResultsFormatsTable(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{
				EvalID:   "e1",
				EvalName: "My Eval",
				Treatments: []result.TreatmentResult{
					{
						Name:      "control",
						IsControl: true,
						Runs: []result.RunResult{
							{Sample: 1, CostUSD: 0.0512, DurationMs: 2300},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	PrintResults(&buf, sr)
	output := buf.String()

	// Check header.
	if !strings.Contains(output, "EVAL") {
		t.Error("missing EVAL column header")
	}
	if !strings.Contains(output, "TREATMENT") {
		t.Error("missing TREATMENT column header")
	}
	if !strings.Contains(output, "SAMPLE") {
		t.Error("missing SAMPLE column header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("missing STATUS column header")
	}
	if !strings.Contains(output, "COST") {
		t.Error("missing COST column header")
	}
	if !strings.Contains(output, "DURATION") {
		t.Error("missing DURATION column header")
	}

	// Check data row.
	if !strings.Contains(output, "My Eval") {
		t.Error("missing eval name in output")
	}
	if !strings.Contains(output, "control") {
		t.Error("missing treatment name in output")
	}
	if !strings.Contains(output, "$0.0512") {
		t.Errorf("missing cost in output, got:\n%s", output)
	}
	if !strings.Contains(output, "ok") {
		t.Error("missing 'ok' status")
	}
	if !strings.Contains(output, "2.3s") {
		t.Errorf("missing duration in output, got:\n%s", output)
	}
}

func TestPrintResultsShowsErrorStatus(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{
			{
				EvalName: "Failing",
				Treatments: []result.TreatmentResult{
					{
						Name: "t1",
						Runs: []result.RunResult{
							{Sample: 1, Err: errors.New("boom")},
							{Sample: 2, IsError: true, CostUSD: 0.01, DurationMs: 500},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	PrintResults(&buf, sr)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), output)
	}

	if !strings.Contains(lines[1], "error") {
		t.Errorf("first data row should show 'error', got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "failed") {
		t.Errorf("second data row should show 'failed', got: %s", lines[2])
	}
}
