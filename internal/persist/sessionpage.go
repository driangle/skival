package persist

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/sessionlink"
)

// linkSessions renders a static vibeview session page for each run that has a
// transcript, and records the page's path (relative to dir) on the run. It
// mutates sr in place and never fails: runs without a page keep an empty
// SessionPage and reports fall back to the run's SessionID.
func linkSessions(dir string, sr *result.SuiteResult) {
	evalsDir := filepath.Join(dir, "evals")
	for i := range sr.Evals {
		eval := &sr.Evals[i]
		for j := range eval.Variants {
			variantDir := filepath.Join(evalsDir, eval.EvalID, eval.Variants[j].Name)
			for k := range eval.Variants[j].Runs {
				linkRun(dir, variantDir, &eval.Variants[j].Runs[k])
			}
		}
	}
}

// linkRun exports one run's transcript to a static page and, on success, sets
// run.SessionPage to the page path relative to the results-dir root.
func linkRun(dir, variantDir string, run *result.RunResult) {
	// No transcript sidecar was written, so there is nothing to render.
	if run.SessionID == "" || len(run.Conversation) == 0 {
		return
	}

	sidecar := filepath.Join(variantDir, fmt.Sprintf("run-%d.conversation.jsonl", run.Sample))
	outPath := filepath.Join(variantDir, fmt.Sprintf("run-%d.session.html", run.Sample))

	link := sessionlink.Export(context.Background(), sessionlink.Request{
		SidecarPath: sidecar,
		SessionID:   run.SessionID,
		OutPath:     outPath,
	})
	if link.Page == "" {
		return
	}

	if rel, err := filepath.Rel(dir, link.Page); err == nil {
		run.SessionPage = rel
	} else {
		run.SessionPage = link.Page
	}
}
