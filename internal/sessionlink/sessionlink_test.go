package sessionlink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validSession is a minimal two-message Claude transcript the vibeview SDK can
// render (mirrors vibeview's own sessionhtml fixture).
const validSession = `{"type":"user","uuid":"u1","sessionId":"sess-1","timestamp":1700000000000,"message":{"role":"user","content":[{"type":"text","text":"hello world"}]}}
{"type":"assistant","uuid":"a1","sessionId":"sess-1","timestamp":1700000001000,"message":{"role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"text","text":"Hi there!"}],"usage":{"input_tokens":10,"output_tokens":5,"costUSD":0.003}}}
`

// writeSidecar writes a transcript JSONL to a temp file and returns its path.
func writeSidecar(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run-1.conversation.jsonl")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing sidecar: %v", err)
	}
	return path
}

func TestExport_RendersPageFromTranscript(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run-1.session.html")
	link := Export(Request{
		SidecarPath: writeSidecar(t, validSession),
		SessionID:   "sess-1",
		OutPath:     out,
	})

	if link.Page != out {
		t.Errorf("Page = %q, want %q", link.Page, out)
	}
	if link.Hint != "" {
		t.Errorf("Hint = %q, want empty on success", link.Hint)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("expected a session page: %v", err)
	}
	// The SDK embeds session data in a known node; its presence proves we wrote
	// a real vibeview page, not an empty file.
	if !strings.Contains(string(data), "vibeview-export-data") {
		t.Errorf("session page missing vibeview data node")
	}
}

func TestExport_RenderFailureFallsBackToHint(t *testing.T) {
	// A path with no transcript makes the SDK fail; Export must not.
	link := Export(Request{
		SidecarPath: filepath.Join(t.TempDir(), "missing.jsonl"),
		SessionID:   "sess-9",
		OutPath:     filepath.Join(t.TempDir(), "out.html"),
	})

	if link.Page != "" {
		t.Errorf("Page = %q, want empty on failure", link.Page)
	}
	if link.Hint != "vibeview show sess-9" {
		t.Errorf("Hint = %q, want fallback command", link.Hint)
	}
}

func TestExport_NoSessionIDYieldsEmptyLink(t *testing.T) {
	link := Export(Request{SidecarPath: filepath.Join(t.TempDir(), "missing.jsonl")})

	if link.Page != "" || link.Hint != "" {
		t.Errorf("want empty Link with no session id, got %+v", link)
	}
}
