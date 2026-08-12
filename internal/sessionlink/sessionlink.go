// Package sessionlink turns a persisted agent transcript into a static session
// page rendered by the external `vibeview` CLI, degrading to a copy-pasteable
// hint when vibeview is unavailable. It never fails the caller: a missing binary
// or a failed export yields a fallback hint, not an error.
package sessionlink

import (
	"context"
	"log/slog"
	"os/exec"
)

// binary is the vibeview executable name, resolved via PATH.
const binary = "vibeview"

// Request describes one session to render.
type Request struct {
	SidecarPath string // path to the run's conversation JSONL (vibeview input)
	SessionID   string // agent session id, used for the fallback hint
	OutPath     string // where the static HTML session page should be written
}

// Link is the outcome of an export attempt. Exactly one field is set: Page when
// a static page was produced, otherwise Hint with a command the user can run.
type Link struct {
	Page string // path to the generated static HTML page
	Hint string // fallback command, e.g. "vibeview show <id>"
}

// Export renders req.SidecarPath to a static HTML page at req.OutPath using
// vibeview. If vibeview is not on PATH or the export fails, it returns a Link
// carrying a fallback hint instead.
func Export(ctx context.Context, req Request) Link {
	path, err := exec.LookPath(binary)
	if err != nil {
		return fallback(req.SessionID)
	}

	cmd := exec.CommandContext(ctx, path, "export", req.SidecarPath, "--format", "html", "--out", req.OutPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("vibeview export failed; falling back to session hint",
			"session_id", req.SessionID, "error", err, "output", string(out))
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
