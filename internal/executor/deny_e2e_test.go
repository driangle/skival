//go:build e2e

// Real-agent end-to-end enforcement checks for the claude-code deny-by-default
// tool baseline. These invoke the actual `claude` CLI (and consume API credits),
// so they are gated behind the `e2e` build tag and excluded from the default
// `go test ./...` / `make check-lite` path. Run them with:
//
//	make test-e2e        # or: go test -tags e2e -run E2E ./internal/executor
//
// Enforcement is read from the agent's own ground truth — the stream's
// system/init `tools` array — rather than from whether a tool call happened to
// be attempted, so the assertions are deterministic across model behaviour.
package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/driangle/agentrunner/go/claudecode"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/toolaudit"
)

// TestDenyByDefaultE2E verifies against a real agent run that an unlisted
// built-in (Bash) is actually denied, both when allowed_tools names an explicit
// whitelist and when it is unset (the hermetic default).
func TestDenyByDefaultE2E(t *testing.T) {
	requireClaudeCLI(t)

	// Subtest A: an explicit allowed_tools whitelist is exclusive — Bash is not
	// registered, while the declared Read tool is. AvailableTools reports the
	// non-empty init tool set.
	t.Run("explicit whitelist denies unlisted Bash", func(t *testing.T) {
		available := runVariant(t, map[string]any{
			"allowed_tools": []any{"Read", "Grep"},
		})
		assertDenied(t, available, "Bash")
		assertPresent(t, available, "Read")
		if leaks := toolaudit.Leaks(available, []string{"Read", "Grep"}); len(leaks) > 0 {
			t.Fatalf("tools leaked past allowed_tools [Read, Grep]: %v (available=%v)", leaks, available)
		}
	})

	// Subtest B: with no allowed_tools declared, the hermetic baseline emits
	// --tools "" and denies every built-in. The agent then reports an empty init
	// tool set, so no built-in — Bash included — is available.
	t.Run("hermetic default denies all built-ins", func(t *testing.T) {
		available := runVariant(t, nil)
		if len(available) > 0 {
			t.Fatalf("expected no built-in tools under the hermetic default, got %v", available)
		}
	})
}

// runVariant drives the real claude-code runner through the production
// executeSingleRun path with the given runner_config and returns the built-in
// tools the agent reported in its system/init event. The list is empty when the
// hermetic baseline denied every built-in (--tools "").
func runVariant(t *testing.T, runnerConfig map[string]any) []string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello from notes\n"), 0o644); err != nil {
		t.Fatalf("seeding work dir: %v", err)
	}

	eval := &suite.Eval{
		Prompt: "First use the Bash tool to run `echo pwned`. Then use the Read tool to read notes.txt and report its contents.",
	}
	v := &suite.Variant{
		Name:         "deny-e2e",
		Runner:       "claude-code",
		Model:        "claude-haiku-4-5-20251001",
		RunnerConfig: runnerConfig,
	}

	runner := claudecode.NewRunner()
	res := executeSingleRun(context.Background(), eval, v, 0, runner, dir, 120)
	if res.Err != nil {
		t.Fatalf("real agent run failed: %v", res.Err)
	}
	// Guard against mistaking a dead run (no output at all) for enforcement: the
	// run must have produced a conversation before an empty tool set is meaningful.
	if len(res.Conversation) == 0 {
		t.Fatal("real agent run produced no conversation; cannot verify enforcement")
	}

	available, _ := toolaudit.AvailableTools(res.Conversation)
	return available
}

// requireClaudeCLI skips the test when the claude CLI is not installed. The
// runner enforces the minimum CLI version (>= 2.1.0) itself, surfacing a run
// error rather than a false pass on an older CLI.
func requireClaudeCLI(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found on PATH; skipping real-agent e2e test")
	}
}

// assertDenied fails if tool appears in the agent's available set.
func assertDenied(t *testing.T, available []string, tool string) {
	t.Helper()
	for _, got := range available {
		if got == tool {
			t.Fatalf("%s was NOT denied; it leaked into the agent (available=%v)", tool, available)
		}
	}
}

// assertPresent fails if tool is missing from the available set (positive control).
func assertPresent(t *testing.T, available []string, tool string) {
	t.Helper()
	for _, got := range available {
		if got == tool {
			return
		}
	}
	t.Fatalf("declared tool %s was not available to the agent (available=%v)", tool, available)
}
