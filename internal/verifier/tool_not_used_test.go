package verifier

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func nestedToolUse(name string) json.RawMessage {
	return json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"` + name + `","input":{}}]}}`)
}

func flatToolUse(name string) json.RawMessage {
	return json.RawMessage(`{"type":"tool_use","name":"` + name + `","input":{}}`)
}

func runToolNotUsed(forbidden []string, conv []json.RawMessage) VerifyResult {
	v := &ToolNotUsedVerifier{Forbidden: forbidden}
	return v.Verify(context.Background(), VerifyInput{Conversation: conv})
}

func TestToolNotUsed_PassWhenForbiddenAbsent_Nested(t *testing.T) {
	conv := []json.RawMessage{nestedToolUse("Read"), nestedToolUse("Grep")}
	got := runToolNotUsed([]string{"Bash", "TaskCreate"}, conv)
	if !got.Pass {
		t.Fatalf("expected pass, got fail: %s", got.Reason)
	}
}

func TestToolNotUsed_PassWhenForbiddenAbsent_Flat(t *testing.T) {
	conv := []json.RawMessage{flatToolUse("read_file"), flatToolUse("run")}
	got := runToolNotUsed([]string{"write_file"}, conv)
	if !got.Pass {
		t.Fatalf("expected pass, got fail: %s", got.Reason)
	}
}

func TestToolNotUsed_FailWhenForbiddenUsed_Nested(t *testing.T) {
	conv := []json.RawMessage{nestedToolUse("Read"), nestedToolUse("TaskCreate"), nestedToolUse("TaskCreate")}
	got := runToolNotUsed([]string{"Skill", "TaskCreate"}, conv)
	if got.Pass {
		t.Fatal("expected fail when a forbidden tool is used")
	}
	if !strings.Contains(got.Reason, "TaskCreate ×2") {
		t.Errorf("reason should name the tool and count, got: %q", got.Reason)
	}
	if strings.Contains(got.Reason, "Skill") {
		t.Errorf("reason should not list an unused forbidden tool, got: %q", got.Reason)
	}
}

func TestToolNotUsed_FailWhenForbiddenUsed_Flat(t *testing.T) {
	conv := []json.RawMessage{flatToolUse("read_file"), flatToolUse("run")}
	got := runToolNotUsed([]string{"run"}, conv)
	if got.Pass {
		t.Fatal("expected fail when a forbidden tool is used")
	}
	if !strings.Contains(got.Reason, "run ×1") {
		t.Errorf("reason should name the tool and count, got: %q", got.Reason)
	}
}

func TestToolNotUsed_MatchesScopedInvocationByBaseName(t *testing.T) {
	// A forbidden base name "Bash" must catch a scoped "Bash(git:*)" invocation.
	conv := []json.RawMessage{nestedToolUse("Bash(git:*)")}
	got := runToolNotUsed([]string{"Bash"}, conv)
	if got.Pass {
		t.Fatal("expected fail: scoped Bash invocation should match forbidden base name")
	}
}

func TestToolNotUsed_PassWhenNoToolActivity(t *testing.T) {
	if got := runToolNotUsed([]string{"Bash"}, nil); !got.Pass {
		t.Fatalf("empty conversation should pass, got: %s", got.Reason)
	}
}
