package report

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

func TestWriteJSON_ValidJSON(t *testing.T) {
	sr := &result.SuiteResult{
		Description: "Test",
		StartedAt:   time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "fizzbuzz",
			Treatments: []result.TreatmentResult{{
				Name:      "control",
				IsControl: true,
				Runs: []result.RunResult{
					{Sample: 1, CostUSD: 0.50, DurationMs: 2000, Pass: boolPtr(true)},
				},
				Aggregate: &result.Aggregate{
					MedianCostUSD: 0.50, MinCostUSD: 0.50, MaxCostUSD: 0.50,
					MedianDurationMs: 2000, MinDurationMs: 2000, MaxDurationMs: 2000,
					Pass: boolPtr(true),
				},
			}},
		}},
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, sr)
	if err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed["description"] != "Test" {
		t.Errorf("unexpected description: %v", parsed["description"])
	}

	evals, ok := parsed["evals"].([]interface{})
	if !ok || len(evals) != 1 {
		t.Fatal("expected 1 eval")
	}
}

func TestWriteJSON_Rankings(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{
				{Name: "a", Runs: []result.RunResult{{CostUSD: 1.0, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "b", Runs: []result.RunResult{{CostUSD: 5.0, DurationMs: 500, Pass: boolPtr(false)}}},
			},
		}},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, sr); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(buf.Bytes(), &parsed)

	rankings, ok := parsed["rankings"].([]interface{})
	if !ok || len(rankings) != 2 {
		t.Fatalf("expected 2 rankings, got %v", parsed["rankings"])
	}
}

func TestWriteJSON_RunStatus(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Treatments: []result.TreatmentResult{{
				Name: "t",
				Runs: []result.RunResult{
					{Sample: 1, Pass: boolPtr(true)},
					{Sample: 2, Pass: boolPtr(false)},
					{Sample: 3},
				},
			}},
		}},
	}

	var buf bytes.Buffer
	WriteJSON(&buf, sr)

	var parsed struct {
		Evals []struct {
			Treatments []struct {
				Runs []struct {
					Status string `json:"status"`
				} `json:"runs"`
			} `json:"treatments"`
		} `json:"evals"`
	}
	json.Unmarshal(buf.Bytes(), &parsed)

	runs := parsed.Evals[0].Treatments[0].Runs
	if runs[0].Status != "pass" {
		t.Errorf("run 1 status = %q, want pass", runs[0].Status)
	}
	if runs[1].Status != "fail" {
		t.Errorf("run 2 status = %q, want fail", runs[1].Status)
	}
	if runs[2].Status != "ok" {
		t.Errorf("run 3 status = %q, want ok", runs[2].Status)
	}
}
