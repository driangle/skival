package suite

import "testing"

// execSuite builds a minimal valid suite with a single exec variant whose
// runner_config is the given map.
func execSuite(cfg map[string]any) *Suite {
	return &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:     "e1",
				Prompt: "p",
				Verify: []VerifyStep{{Type: "agent_exits_ok"}},
				Variants: []Variant{
					{Name: "baseline", Runner: "exec", RunnerConfig: cfg},
				},
			},
		},
	}
}

func TestValidate_ExecVariantNeedsNoModel(t *testing.T) {
	// A valid exec variant omits model entirely.
	s := execSuite(map[string]any{"command": []any{"python", "agent.py"}})
	if err := validate(s); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_ExecRequiresCommand(t *testing.T) {
	s := execSuite(map[string]any{"prompt_via": "stdin"})
	err := validate(s)
	assertValidationContains(t, err, "command is required")
}

func TestValidate_ExecPromptViaEnum(t *testing.T) {
	s := execSuite(map[string]any{"command": []any{"x"}, "prompt_via": "telepathy"})
	err := validate(s)
	assertValidationContains(t, err, "prompt_via must be one of")
}

func TestValidate_ExecArgFileNeedsPlaceholder(t *testing.T) {
	s := execSuite(map[string]any{"command": []any{"cat"}, "prompt_via": "arg-file"})
	err := validate(s)
	assertValidationContains(t, err, "{prompt_file}")
}

func TestValidate_ExecInvalidConfigShape(t *testing.T) {
	s := execSuite(map[string]any{"command": "not-a-list"})
	err := validate(s)
	assertValidationContains(t, err, "invalid exec runner_config")
}
