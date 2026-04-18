package report

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	err := WriteJSON(&buf, sr, DefaultWeights())
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
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	rankings, ok := parsed["rankings"].([]interface{})
	if !ok || len(rankings) != 2 {
		t.Fatalf("expected 2 rankings, got %v", parsed["rankings"])
	}
}

func TestWriteJSON_RunnerField(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "test",
			Treatments: []result.TreatmentResult{{
				Name:   "ctrl",
				Runner: "claude-code",
				Runs:   []result.RunResult{{Sample: 1, CostUSD: 0.1, DurationMs: 100}},
			}},
		}},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed struct {
		Evals []struct {
			Treatments []struct {
				Runner string `json:"runner"`
			} `json:"treatments"`
		} `json:"evals"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.Evals[0].Treatments[0].Runner != "claude-code" {
		t.Errorf("expected runner %q, got %q", "claude-code", parsed.Evals[0].Treatments[0].Runner)
	}
}

func TestWriteJSON_ModelField(t *testing.T) {
	sr := &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "test",
			Treatments: []result.TreatmentResult{
				{
					Name:   "ctrl",
					Runner: "claude-code",
					Model:  "claude-sonnet-4-6",
					Runs:   []result.RunResult{{Sample: 1, CostUSD: 0.1, DurationMs: 100, Pass: boolPtr(true)}},
				},
				{
					Name:   "var",
					Runner: "ollama",
					Model:  "llama3",
					Runs:   []result.RunResult{{Sample: 1, CostUSD: 0.01, DurationMs: 200, Pass: boolPtr(true)}},
				},
			},
		}},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed struct {
		Evals []struct {
			Treatments []struct {
				Model string `json:"model"`
			} `json:"treatments"`
		} `json:"evals"`
		Rankings []struct {
			Model string `json:"model"`
		} `json:"rankings"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.Evals[0].Treatments[0].Model != "claude-sonnet-4-6" {
		t.Errorf("expected model %q, got %q", "claude-sonnet-4-6", parsed.Evals[0].Treatments[0].Model)
	}
	if parsed.Evals[0].Treatments[1].Model != "llama3" {
		t.Errorf("expected model %q, got %q", "llama3", parsed.Evals[0].Treatments[1].Model)
	}
	if len(parsed.Rankings) >= 2 && parsed.Rankings[0].Model == "" {
		t.Error("expected model field in rankings")
	}
}

func TestWriteJSON_EvalError(t *testing.T) {
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
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed struct {
		Evals []struct {
			ID    string `json:"id"`
			Error string `json:"error"`
		} `json:"evals"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(parsed.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(parsed.Evals))
	}
	if parsed.Evals[0].Error == "" {
		t.Error("expected error field to be populated")
	}
	if parsed.Evals[0].ID != "broken" {
		t.Errorf("expected id 'broken', got %q", parsed.Evals[0].ID)
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
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var parsed struct {
		Evals []struct {
			Treatments []struct {
				Runs []struct {
					Status string `json:"status"`
				} `json:"runs"`
			} `json:"treatments"`
		} `json:"evals"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

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
