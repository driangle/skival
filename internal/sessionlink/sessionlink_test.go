package sessionlink

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeFakeVibeview installs an executable named "vibeview" in a fresh dir and
// puts that dir first on PATH, so Export resolves it. body is the script body
// after the shebang.
func writeFakeVibeview(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body
	path := filepath.Join(dir, "vibeview")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake vibeview: %v", err)
	}
	t.Setenv("PATH", dir)
}

func TestExport_SuccessReturnsPage(t *testing.T) {
	// Export always invokes: export <sidecar> --format html --out <outPath>,
	// so the out path is the 6th argument.
	writeFakeVibeview(t, `printf '<html></html>' > "$6"`)

	out := filepath.Join(t.TempDir(), "run-1.session.html")
	link := Export(context.Background(), Request{
		SidecarPath: "conversation.jsonl",
		SessionID:   "abc123",
		OutPath:     out,
	})

	if link.Page != out {
		t.Errorf("Page = %q, want %q", link.Page, out)
	}
	if link.Hint != "" {
		t.Errorf("Hint = %q, want empty on success", link.Hint)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected session page to be written: %v", err)
	}
}

func TestExport_CommandFailureFallsBackToHint(t *testing.T) {
	writeFakeVibeview(t, "exit 1\n")

	link := Export(context.Background(), Request{
		SidecarPath: "conversation.jsonl",
		SessionID:   "abc123",
		OutPath:     filepath.Join(t.TempDir(), "out.html"),
	})

	if link.Page != "" {
		t.Errorf("Page = %q, want empty on failure", link.Page)
	}
	if link.Hint != "vibeview show abc123" {
		t.Errorf("Hint = %q, want fallback command", link.Hint)
	}
}

func TestExport_MissingBinaryFallsBackToHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir: no vibeview on PATH

	link := Export(context.Background(), Request{SessionID: "xyz"})

	if link.Page != "" {
		t.Errorf("Page = %q, want empty when vibeview absent", link.Page)
	}
	if link.Hint != "vibeview show xyz" {
		t.Errorf("Hint = %q, want fallback command", link.Hint)
	}
}

func TestExport_NoSessionIDYieldsEmptyLink(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	link := Export(context.Background(), Request{})

	if link.Page != "" || link.Hint != "" {
		t.Errorf("want empty Link with no session id, got %+v", link)
	}
}
