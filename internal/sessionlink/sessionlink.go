// Package sessionlink turns a persisted agent transcript into a static session
// page using the vibeview SDK, which is compiled into skival — no external
// binary is required. It never fails the caller: a render or write error yields
// a fallback hint, not an error.
package sessionlink

import (
	"log/slog"
	"os"

	"github.com/driangle/vibeview/apps/lib/sessionhtml"
)

// Request describes one session to render.
type Request struct {
	SidecarPath string // path to the run's conversation JSONL (vibeview input)
	SessionID   string // agent session id, used for the fallback hint
	OutPath     string // where the static HTML session page should be written
}

// Link is the outcome of a render attempt. Exactly one field is set: Page when a
// static page was produced, otherwise Hint with a command the user can run.
type Link struct {
	Page string // path to the generated static HTML page
	Hint string // fallback command, e.g. "vibeview show <id>"
}

// Export renders req.SidecarPath to a self-contained HTML page at req.OutPath via
// the vibeview SDK. On a render or write error it returns a Link carrying a
// fallback hint instead.
func Export(req Request) Link {
	page, err := sessionhtml.RenderSessionHTML(sessionhtml.Request{
		Session:     req.SidecarPath,
		CostEnabled: true,
	})
	if err != nil {
		slog.Warn("rendering session page failed; falling back to session hint",
			"session_id", req.SessionID, "error", err)
		return fallback(req.SessionID)
	}

	if err := os.WriteFile(req.OutPath, page, 0o644); err != nil {
		slog.Warn("writing session page failed; falling back to session hint",
			"session_id", req.SessionID, "error", err)
		return fallback(req.SessionID)
	}

	return Link{Page: req.OutPath}
}

// fallback builds the hint shown when a static page could not be produced.
func fallback(sessionID string) Link {
	if sessionID == "" {
		return Link{}
	}
	return Link{Hint: "vibeview show " + sessionID}
}
