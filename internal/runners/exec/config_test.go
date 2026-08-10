package exec

import (
	"reflect"
	"testing"
)

func TestParseConfig_FullShape(t *testing.T) {
	m := map[string]any{
		"command":     []any{"python", "agent.py"},
		"prompt_via":  "stdin",
		"prompt_env":  "MY_PROMPT",
		"events_path": "${SKIVAL_RUN_DIR}/events.jsonl",
	}
	cfg, err := ParseConfig(m)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	want := Config{
		Command:    []string{"python", "agent.py"},
		PromptVia:  "stdin",
		PromptEnv:  "MY_PROMPT",
		EventsPath: "${SKIVAL_RUN_DIR}/events.jsonl",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("cfg = %+v, want %+v", cfg, want)
	}
}

func TestParseConfig_RejectsNonStringCommand(t *testing.T) {
	if _, err := ParseConfig(map[string]any{"command": []any{"python", 3}}); err == nil {
		t.Error("expected error for non-string command element")
	}
	if _, err := ParseConfig(map[string]any{"command": "python agent.py"}); err == nil {
		t.Error("expected error for scalar command")
	}
}

func TestParseConfig_RejectsNonStringFields(t *testing.T) {
	if _, err := ParseConfig(map[string]any{"command": []any{"x"}, "prompt_via": 1}); err == nil {
		t.Error("expected error for non-string prompt_via")
	}
}

func TestValidate_RequiresCommand(t *testing.T) {
	errs := Config{}.Validate()
	if len(errs) == 0 {
		t.Error("expected error for missing command")
	}
}

func TestValidate_PromptViaEnum(t *testing.T) {
	for _, mode := range []string{"", PromptViaStdin, PromptViaEnv} {
		if errs := (Config{Command: []string{"x"}, PromptVia: mode}).Validate(); len(errs) != 0 {
			t.Errorf("mode %q: unexpected errors %v", mode, errs)
		}
	}
	if errs := (Config{Command: []string{"x"}, PromptVia: "bogus"}).Validate(); len(errs) == 0 {
		t.Error("expected error for unknown prompt_via")
	}
}

func TestValidate_ArgFileRequiresPlaceholder(t *testing.T) {
	// Missing placeholder is an error.
	if errs := (Config{Command: []string{"cat"}, PromptVia: PromptViaArgFile}).Validate(); len(errs) == 0 {
		t.Error("expected error when arg-file mode lacks {prompt_file}")
	}
	// Present placeholder is valid.
	cfg := Config{Command: []string{"cat", PromptFilePlaceholder}, PromptVia: PromptViaArgFile}
	if errs := cfg.Validate(); len(errs) != 0 {
		t.Errorf("unexpected errors %v", errs)
	}
}
