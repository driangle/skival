package persist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

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
					Conversation: []json.RawMessage{json.RawMessage(`{"type":"user"}`)},
				}},
			}},
		}},
	}
}

// installFakeVibeview puts a fake `vibeview export` on PATH that writes its
// --out target, so linkSessions produces a page without the real binary.
func installFakeVibeview(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '<html></html>' > \"$6\"\n"
	if err := os.WriteFile(filepath.Join(dir, "vibeview"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake vibeview: %v", err)
	}
	t.Setenv("PATH", dir)
}

func TestSave_LinkSessionsPersistsSessionPage(t *testing.T) {
	installFakeVibeview(t)
	sr := suiteWithConversation()

	outDir, err := Save(t.TempDir(), sr, defaultWeights(), SaveOptions{LinkSessions: true})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantRel := filepath.Join("evals", "eval1", "baseline", "run-1.session.html")

	// The page file exists on disk...
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

func TestSave_LinkSessionsFallsBackWhenVibeviewAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no vibeview on PATH
	sr := suiteWithConversation()

	outDir, err := Save(t.TempDir(), sr, defaultWeights(), SaveOptions{LinkSessions: true})
	if err != nil {
		t.Fatalf("Save must not fail when vibeview is absent: %v", err)
	}

	// No page produced, but the session id survives for the report's fallback hint.
	if got := sr.Evals[0].Variants[0].Runs[0].SessionPage; got != "" {
		t.Errorf("SessionPage = %q, want empty on fallback", got)
	}
	if _, err := os.Stat(filepath.Join(outDir, "evals", "eval1", "baseline", "run-1.session.html")); !os.IsNotExist(err) {
		t.Errorf("expected no session page on fallback, stat err = %v", err)
	}
	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.Evals[0].Variants[0].Runs[0].SessionID; got != "sess-abc" {
		t.Errorf("loaded SessionID = %q, want preserved", got)
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
