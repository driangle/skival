package persist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
)

func defaultWeights() report.Weights { return report.DefaultWeights() }

func boolPtr(b bool) *bool { return &b }

func makeSuiteResult() *result.SuiteResult {
	cv := 0.1
	return &result.SuiteResult{
		Description: "test suite",
		StartedAt:   time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
		Evals: []result.EvalResult{{
			EvalID:   "eval1",
			EvalName: "fizzbuzz",
			Variants: []result.VariantResult{{
				Name:      "control",
				IsControl: true,
				Runs: []result.RunResult{
					{Sample: 1, Text: "hello", CostUSD: 0.50, DurationMs: 2000, Pass: boolPtr(true)},
					{Sample: 2, Text: "world", CostUSD: 0.60, DurationMs: 2500, Pass: boolPtr(true)},
				},
				Aggregate: &result.Aggregate{
					MedianCostUSD: 0.55, MinCostUSD: 0.50, MaxCostUSD: 0.60,
					MedianDurationMs: 2250, MinDurationMs: 2000, MaxDurationMs: 2500,
					CostCV: &cv, Pass: boolPtr(true),
				},
			}},
		}},
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify directory structure
	if _, err := os.Stat(filepath.Join(outDir, "summary.json")); err != nil {
		t.Error("missing summary.json")
	}
	if _, err := os.Stat(filepath.Join(outDir, "summary.md")); err != nil {
		t.Error("missing summary.md")
	}
	if _, err := os.Stat(filepath.Join(outDir, "evals", "eval1", "control", "run-1.json")); err != nil {
		t.Error("missing run-1.json")
	}
	if _, err := os.Stat(filepath.Join(outDir, "evals", "eval1", "control", "run-2.json")); err != nil {
		t.Error("missing run-2.json")
	}
	if _, err := os.Stat(filepath.Join(outDir, "evals", "eval1", "control", "aggregate.json")); err != nil {
		t.Error("missing aggregate.json")
	}

	// Load and verify
	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Description != sr.Description {
		t.Errorf("description = %q, want %q", loaded.Description, sr.Description)
	}
	if len(loaded.Evals) != 1 {
		t.Fatalf("expected 1 eval, got %d", len(loaded.Evals))
	}
	if len(loaded.Evals[0].Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(loaded.Evals[0].Variants))
	}

	treat := loaded.Evals[0].Variants[0]
	if treat.Name != "control" {
		t.Errorf("variant name = %q, want control", treat.Name)
	}
	if len(treat.Runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(treat.Runs))
	}
	if treat.Runs[0].CostUSD != 0.50 {
		t.Errorf("run 1 cost = %f, want 0.50", treat.Runs[0].CostUSD)
	}
	if treat.Runs[0].Pass == nil || !*treat.Runs[0].Pass {
		t.Error("run 1 pass should be true")
	}
	if treat.Runs[0].Text != "hello" {
		t.Errorf("run 1 text = %q, want hello", treat.Runs[0].Text)
	}
}

func TestSave_TimestampedDir(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	expected := filepath.Join(tmpDir, "20260319-100000")
	if outDir != expected {
		t.Errorf("outDir = %q, want %q", outDir, expected)
	}
}

func TestSave_RunJSONContent(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()

	outDir, _ := Save(tmpDir, sr, defaultWeights())

	data, err := os.ReadFile(filepath.Join(outDir, "evals", "eval1", "control", "run-1.json"))
	if err != nil {
		t.Fatal(err)
	}

	var rj runJSON
	if err := json.Unmarshal(data, &rj); err != nil {
		t.Fatal(err)
	}
	if rj.Sample != 1 {
		t.Errorf("sample = %d, want 1", rj.Sample)
	}
	if rj.CostUSD != 0.50 {
		t.Errorf("cost = %f, want 0.50", rj.CostUSD)
	}
	if rj.Pass == nil || !*rj.Pass {
		t.Error("pass should be true")
	}
}

func TestSave_SummaryJSON(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()

	outDir, _ := Save(tmpDir, sr, defaultWeights())

	data, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}

	var summary summaryJSON
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Description != "test suite" {
		t.Errorf("description = %q", summary.Description)
	}
}

func TestLoad_NonExistentDir(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent dir")
	}
}

func TestSave_WithConversations(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()
	sr.Evals[0].Variants[0].Runs[0].Conversation = []json.RawMessage{
		json.RawMessage(`{"role":"assistant","text":"hello"}`),
		json.RawMessage(`{"role":"user","text":"hi"}`),
	}
	sr.Evals[0].Variants[0].Runs[0].JudgeConversation = []json.RawMessage{
		json.RawMessage(`{"role":"assistant","text":"PASS: ok"}`),
	}

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	convPath := filepath.Join(outDir, "evals", "eval1", "control", "run-1.conversation.jsonl")
	if _, err := os.Stat(convPath); err != nil {
		t.Errorf("missing conversation.jsonl: %v", err)
	}
	judgePath := filepath.Join(outDir, "evals", "eval1", "control", "run-1.judge.jsonl")
	if _, err := os.Stat(judgePath); err != nil {
		t.Errorf("missing judge.jsonl: %v", err)
	}

	// run-2 has no conversations — files should not exist.
	noConvPath := filepath.Join(outDir, "evals", "eval1", "control", "run-2.conversation.jsonl")
	if _, err := os.Stat(noConvPath); err == nil {
		t.Error("run-2.conversation.jsonl should not exist")
	}
}

func TestSave_WithoutConversations_NoJSONLFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	convPath := filepath.Join(outDir, "evals", "eval1", "control", "run-1.conversation.jsonl")
	if _, err := os.Stat(convPath); err == nil {
		t.Error("conversation.jsonl should not exist when no conversations")
	}
}

func TestSaveAndLoad_ConversationRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()
	sr.Evals[0].Variants[0].Runs[0].Conversation = []json.RawMessage{
		json.RawMessage(`{"role":"assistant","text":"hello"}`),
	}
	sr.Evals[0].Variants[0].Runs[0].JudgeConversation = []json.RawMessage{
		json.RawMessage(`{"role":"assistant","text":"PASS"}`),
	}

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	run := loaded.Evals[0].Variants[0].Runs[0]
	if len(run.Conversation) != 1 {
		t.Fatalf("expected 1 conversation message, got %d", len(run.Conversation))
	}
	if string(run.Conversation[0]) != `{"role":"assistant","text":"hello"}` {
		t.Errorf("conversation mismatch: %s", run.Conversation[0])
	}
	if len(run.JudgeConversation) != 1 {
		t.Fatalf("expected 1 judge conversation message, got %d", len(run.JudgeConversation))
	}
	if string(run.JudgeConversation[0]) != `{"role":"assistant","text":"PASS"}` {
		t.Errorf("judge conversation mismatch: %s", run.JudgeConversation[0])
	}
}

func TestLoad_BackwardsCompat_NoJSONLFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult()

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	run := loaded.Evals[0].Variants[0].Runs[0]
	if run.Conversation != nil {
		t.Error("expected nil conversation for legacy data")
	}
	if run.JudgeConversation != nil {
		t.Error("expected nil judge conversation for legacy data")
	}
}

func TestSave_EmptySuite(t *testing.T) {
	tmpDir := t.TempDir()
	sr := &result.SuiteResult{
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
	}

	outDir, err := Save(tmpDir, sr, defaultWeights())
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded.Evals) != 0 {
		t.Errorf("expected 0 evals, got %d", len(loaded.Evals))
	}
}
