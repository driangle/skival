package verifier

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSummarizeToolActivity_Empty(t *testing.T) {
	if got := SummarizeToolActivity(nil); got != "" {
		t.Errorf("nil conversation: got %q, want empty", got)
	}
	if got := SummarizeToolActivity([]json.RawMessage{}); got != "" {
		t.Errorf("empty conversation: got %q, want empty", got)
	}
}

func TestSummarizeToolActivity_ToolUse(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"gh pr list"}}]}}`),
	}
	got := SummarizeToolActivity(msgs)
	if !strings.Contains(got, "tool_use Bash") {
		t.Errorf("missing tool_use line: %q", got)
	}
	if !strings.Contains(got, "gh pr list") {
		t.Errorf("missing tool input: %q", got)
	}
}

func TestSummarizeToolActivity_ToolResultString(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"user","message":{"content":[{"type":"tool_result","content":"Done"}]}}`),
	}
	got := SummarizeToolActivity(msgs)
	if !strings.Contains(got, "tool_result: Done") {
		t.Errorf("string tool_result not rendered: %q", got)
	}
}

func TestSummarizeToolActivity_ToolResultArray(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"user","message":{"content":[{"type":"tool_result","content":[{"type":"text","text":"line one"},{"type":"text","text":"line two"}]}]}}`),
	}
	got := SummarizeToolActivity(msgs)
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Errorf("array tool_result not flattened: %q", got)
	}
}

func TestSummarizeToolActivity_SkipsNonTool(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"},{"type":"thinking","thinking":"hmm"}]}}`),
	}
	if got := SummarizeToolActivity(msgs); got != "" {
		t.Errorf("expected empty for text/thinking only: %q", got)
	}
}

func TestSummarizeToolActivity_SkipsMalformed(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`not json`),
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`),
	}
	got := SummarizeToolActivity(msgs)
	if !strings.Contains(got, "tool_use Bash") {
		t.Errorf("expected tool_use after malformed skip: %q", got)
	}
}

func TestSummarizeToolActivity_TruncatesLongBlock(t *testing.T) {
	long := strings.Repeat("x", maxToolBlockLen*2)
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"user","message":{"content":[{"type":"tool_result","content":"` + long + `"}]}}`),
	}
	got := SummarizeToolActivity(msgs)
	if !strings.Contains(got, "...") {
		t.Errorf("expected ellipsis truncation, got: %q", got[:min(len(got), 200)])
	}
	if len(got) > maxToolBlockLen+100 {
		t.Errorf("output not truncated: len=%d", len(got))
	}
}

func TestSummarizeToolActivity_MultipleMessages(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`),
		json.RawMessage(`{"type":"user","message":{"content":[{"type":"tool_result","content":"file.txt"}]}}`),
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"path":"/tmp/x"}}]}}`),
	}
	got := SummarizeToolActivity(msgs)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "-> tool_use Bash") {
		t.Errorf("line 0 wrong: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "<- tool_result: file.txt") {
		t.Errorf("line 1 wrong: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "-> tool_use Read") {
		t.Errorf("line 2 wrong: %q", lines[2])
	}
}
