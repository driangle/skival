package executor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/color"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
)

func initRun(tools ...string) result.RunResult {
	line, _ := json.Marshal(map[string]any{"type": "system", "subtype": "init", "tools": tools})
	return result.RunResult{Conversation: []json.RawMessage{line}}
}

func variantWithAllowed(allowed ...string) *suite.Variant {
	cfg := map[string]any{}
	if len(allowed) > 0 {
		cfg["allowed_tools"] = allowed
	}
	return &suite.Variant{Name: "restricted", RunnerConfig: cfg}
}

func runPreflight(t *testing.T, v *suite.Variant, run result.RunResult) string {
	t.Helper()
	color.SetEnabled(false)
	var buf bytes.Buffer
	preflightToolCheck(v, run, newProgress(&buf))
	return buf.String()
}

func TestPreflightWarnsOnLeak(t *testing.T) {
	out := runPreflight(t, variantWithAllowed("Read", "Bash"), initRun("Read", "Bash", "Skill", "ToolSearch"))
	if !strings.Contains(out, "beyond allowed_tools") {
		t.Fatalf("expected a leak warning, got %q", out)
	}
	for _, tool := range []string{"Skill", "ToolSearch"} {
		if !strings.Contains(out, tool) {
			t.Fatalf("expected warning to name %q, got %q", tool, out)
		}
	}
	if strings.Contains(out, "Read") || strings.Contains(out, "Bash") {
		t.Fatalf("allowed tools should not be listed as extras, got %q", out)
	}
}

func TestPreflightSilentWhenSubset(t *testing.T) {
	out := runPreflight(t, variantWithAllowed("Read", "Bash", "Skill"), initRun("Read", "Bash"))
	if out != "" {
		t.Fatalf("expected no warning when available is a subset, got %q", out)
	}
}

func TestPreflightSilentWhenNoAllowedDeclared(t *testing.T) {
	out := runPreflight(t, variantWithAllowed(), initRun("Read", "Bash", "Skill"))
	if out != "" {
		t.Fatalf("expected no warning when allowed_tools is unset, got %q", out)
	}
}

func TestPreflightSilentWhenNoInitEvent(t *testing.T) {
	run := result.RunResult{Conversation: []json.RawMessage{json.RawMessage(`{"type":"assistant"}`)}}
	out := runPreflight(t, variantWithAllowed("Read"), run)
	if out != "" {
		t.Fatalf("expected no warning when the runner emits no init tool list, got %q", out)
	}
}
