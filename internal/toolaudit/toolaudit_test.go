package toolaudit

import (
	"encoding/json"
	"reflect"
	"testing"
)

// raw builds a conversation from JSON line literals.
func raw(lines ...string) []json.RawMessage {
	conv := make([]json.RawMessage, len(lines))
	for i, l := range lines {
		conv[i] = json.RawMessage(l)
	}
	return conv
}

func TestAvailableToolsExtractsFromInit(t *testing.T) {
	conv := raw(
		`{"type":"system","subtype":"other"}`,
		`{"type":"system","subtype":"init","tools":["Read","Bash","Skill"],"session_id":"abc"}`,
		`{"type":"assistant","message":{}}`,
	)
	tools, ok := AvailableTools(conv)
	if !ok {
		t.Fatal("expected ok=true when an init event carries tools")
	}
	want := []string{"Read", "Bash", "Skill"}
	if !reflect.DeepEqual(tools, want) {
		t.Fatalf("tools = %v, want %v", tools, want)
	}
}

func TestAvailableToolsUsesFirstInitEvent(t *testing.T) {
	conv := raw(
		`{"type":"system","subtype":"init","tools":["Read"]}`,
		`{"type":"system","subtype":"init","tools":["Bash"]}`,
	)
	tools, ok := AvailableTools(conv)
	if !ok || !reflect.DeepEqual(tools, []string{"Read"}) {
		t.Fatalf("tools = %v ok = %v, want [Read] true", tools, ok)
	}
}

func TestAvailableToolsAcceptsObjectEntries(t *testing.T) {
	conv := raw(`{"type":"system","subtype":"init","tools":[{"name":"Read"},{"name":"Bash"}]}`)
	tools, ok := AvailableTools(conv)
	if !ok || !reflect.DeepEqual(tools, []string{"Read", "Bash"}) {
		t.Fatalf("tools = %v ok = %v, want [Read Bash] true", tools, ok)
	}
}

func TestAvailableToolsNoInit(t *testing.T) {
	cases := map[string][]json.RawMessage{
		"empty":       nil,
		"no init":     raw(`{"type":"assistant","message":{}}`),
		"no tools":    raw(`{"type":"system","subtype":"init","session_id":"x"}`),
		"empty tools": raw(`{"type":"system","subtype":"init","tools":[]}`),
		"malformed":   raw(`not json`),
	}
	for name, conv := range cases {
		t.Run(name, func(t *testing.T) {
			tools, ok := AvailableTools(conv)
			if ok || tools != nil {
				t.Fatalf("expected (nil, false), got (%v, %v)", tools, ok)
			}
		})
	}
}

func TestLeaksReportsExtras(t *testing.T) {
	available := []string{"Read", "Bash", "Skill", "ToolSearch"}
	allowed := []string{"Read", "Bash"}
	got := Leaks(available, allowed)
	want := []string{"Skill", "ToolSearch"} // sorted
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Leaks = %v, want %v", got, want)
	}
}

func TestLeaksNoneWhenSubset(t *testing.T) {
	if got := Leaks([]string{"Read", "Bash"}, []string{"Read", "Bash", "Grep"}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestLeaksNilWhenNoAllowedDeclared(t *testing.T) {
	if got := Leaks([]string{"Read", "Bash"}, nil); got != nil {
		t.Fatalf("expected nil when allowed is empty, got %v", got)
	}
}

func TestLeaksScopedEntryCoversBaseTool(t *testing.T) {
	// A scoped allow entry like Bash(git:*) must cover the unscoped "Bash" the
	// init event reports, so it is not falsely flagged as a leak.
	got := Leaks([]string{"Bash", "Skill"}, []string{"Bash(git:*)", "Read"})
	if !reflect.DeepEqual(got, []string{"Skill"}) {
		t.Fatalf("Leaks = %v, want [Skill]", got)
	}
}

func TestLeaksResultSorted(t *testing.T) {
	got := Leaks([]string{"Zed", "Alpha", "Mid"}, []string{"Read"})
	want := []string{"Alpha", "Mid", "Zed"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Leaks = %v, want %v", got, want)
	}
}
