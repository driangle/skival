package persist

import (
	"encoding/json"
	"testing"
)

// TestSaveAndLoad_ToolCountsRecomputed verifies that a loaded run's ToolCounts
// is derived from its persisted conversation sidecar (not from run-N.json, which
// carries no tool field). The count must survive a save/load round-trip.
func TestSaveAndLoad_ToolCountsRecomputed(t *testing.T) {
	sr := makeSuiteResult()
	sr.Evals[0].Variants[0].Runs[0].Conversation = []json.RawMessage{
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{}}]}}`),
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{}},{"type":"tool_use","name":"TaskCreate","input":{}}]}}`),
	}

	outDir, err := Save(t.TempDir(), sr, defaultWeights(), SaveOptions{})
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	got := loaded.Evals[0].Variants[0].Runs[0].ToolCounts
	want := map[string]int{"Read": 2, "TaskCreate": 1}
	if len(got) != len(want) {
		t.Fatalf("ToolCounts = %v, want %v", got, want)
	}
	for name, n := range want {
		if got[name] != n {
			t.Errorf("ToolCounts[%q] = %d, want %d", name, got[name], n)
		}
	}
}

// runsWithoutConversation are the round-trip runs from makeSuiteResult, which
// carry no conversation — a loaded run must then have empty ToolCounts.
func TestSaveAndLoad_ToolCountsEmptyWithoutConversation(t *testing.T) {
	outDir, err := Save(t.TempDir(), makeSuiteResult(), defaultWeights(), SaveOptions{})
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if tc := loaded.Evals[0].Variants[0].Runs[0].ToolCounts; len(tc) != 0 {
		t.Errorf("expected empty ToolCounts, got %v", tc)
	}
}
