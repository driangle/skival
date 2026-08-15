package verifier

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCountToolUses_Empty(t *testing.T) {
	if got := CountToolUses(nil); len(got) != 0 {
		t.Errorf("nil conversation: got %v, want empty", got)
	}
	if got := CountToolUses([]json.RawMessage{}); len(got) != 0 {
		t.Errorf("empty conversation: got %v, want empty", got)
	}
}

func TestCountToolUses_NestedClaudeCode(t *testing.T) {
	// Nested claude-code shape: tool_use blocks under message.content. Repeated
	// tools accumulate; tool_result and non-tool blocks are ignored.
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{}},{"type":"text","text":"hi"}]}}`),
		json.RawMessage(`{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}`),
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{}},{"type":"tool_use","name":"TaskCreate","input":{}}]}}`),
	}
	got := CountToolUses(msgs)
	want := map[string]int{"Read": 2, "TaskCreate": 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("counts = %v, want %v", got, want)
	}
}

func TestCountToolUses_FlatExecEvents(t *testing.T) {
	// Flat exec-runner shape: type/name/input at the top level.
	msgs := []json.RawMessage{
		json.RawMessage(`{"type":"tool_use","name":"read_file","input":{"path":"a"}}`),
		json.RawMessage(`{"type":"tool_result","content":"body"}`),
		json.RawMessage(`{"type":"tool_use","name":"read_file","input":{"path":"b"}}`),
		json.RawMessage(`{"type":"tool_use","name":"run","input":{}}`),
		json.RawMessage(`{"type":"message","role":"assistant","content":"noise"}`),
	}
	got := CountToolUses(msgs)
	want := map[string]int{"read_file": 2, "run": 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("counts = %v, want %v", got, want)
	}
}

func TestCountToolUses_SkipsMalformedAndEmptyNames(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`not json`),
		json.RawMessage(`{"type":"tool_use","input":{}}`), // no name
		json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{}}]}}`),
	}
	got := CountToolUses(msgs)
	want := map[string]int{"Bash": 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("counts = %v, want %v", got, want)
	}
}
