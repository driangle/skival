package persist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

// validConversation is a minimal Claude transcript the vibeview SDK renders.
var validConversation = []json.RawMessage{
	json.RawMessage(`{"type":"user","uuid":"u1","sessionId":"sess-abc","timestamp":1700000000000,"message":{"role":"user","content":[{"type":"text","text":"hello"}]}}`),
	json.RawMessage(`{"type":"assistant","uuid":"a1","sessionId":"sess-abc","timestamp":1700000001000,"message":{"role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":3,"output_tokens":2}}}`),
}

// suiteWithConversation returns a one-run suite whose run carries a transcript
// sidecar and a session id, the prerequisites for session linking.
func suiteWithConversation() *result.SuiteResult {
	return &result.SuiteResult{
		StartedAt:  time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 3, 19, 10, 1, 0, 0, time.UTC),
		Evals: []result.EvalResult{{
			EvalID:   "eval1",
			EvalName: "e1",
			Variants: []result.VariantResult{{
				Name: "baseline",
				Runs: []result.RunResult{{
					Sample:       1,
					Pass:         boolPtr(true),
					SessionID:    "sess-abc",
					Conversation: validConversation,
				}},
			}},
		}},
	}
}

func TestSave_LinkSessionsPersistsSessionPage(t *testing.T) {
	sr := suiteWithConversation()

	outDir, err := Save(t.TempDir(), sr, defaultWeights(), SaveOptions{LinkSessions: true})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantRel := filepath.Join("evals", "eval1", "baseline", "run-1.session.html")

	// The page file exists on disk (rendered in-process by the vibeview SDK)...
	if _, err := os.Stat(filepath.Join(outDir, wantRel)); err != nil {
		t.Errorf("expected session page on disk: %v", err)
	}
	// ...the in-memory run was updated (so the immediate report links it)...
	if got := sr.Evals[0].Variants[0].Runs[0].SessionPage; got != wantRel {
		t.Errorf("in-memory SessionPage = %q, want %q", got, wantRel)
	}
	// ...and it round-trips through load so `skival report <dir>` sees it.
	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.Evals[0].Variants[0].Runs[0].SessionPage; got != wantRel {
		t.Errorf("loaded SessionPage = %q, want %q", got, wantRel)
	}
}

func TestSave_NoLinkSessionsRoundTripsPresetSessionPage(t *testing.T) {
	sr := suiteWithConversation()
	sr.Evals[0].Variants[0].Runs[0].SessionPage = "preset/run-1.session.html"

	outDir, err := Save(t.TempDir(), sr, defaultWeights(), SaveOptions{})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.Evals[0].Variants[0].Runs[0].SessionPage; got != "preset/run-1.session.html" {
		t.Errorf("SessionPage did not round-trip: %q", got)
	}
}
