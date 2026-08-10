package suite

import (
	"errors"
	"testing"
)

func TestModelLooksValidForRunner(t *testing.T) {
	tests := []struct {
		model  string
		runner string
		valid  bool
	}{
		{"claude-sonnet-4-6", "claude-code", true},
		{"claude-opus-4-6", "claude-code", true},
		{"gpt-4o", "claude-code", false},
		{"llama3", "claude-code", false},
		{"gpt-4o", "codex", true},
		{"o1-mini", "codex", true},
		{"o3-mini", "codex", true},
		{"codex-mini", "codex", true},
		{"claude-sonnet-4-6", "codex", false},
		{"llama3", "ollama", true},
		{"claude-sonnet-4-6", "ollama", true}, // ollama accepts anything
		{"any-model", "aider", true},          // aider accepts anything
	}

	for _, tt := range tests {
		got := modelLooksValidForRunner(tt.model, tt.runner)
		if got != tt.valid {
			t.Errorf("modelLooksValidForRunner(%q, %q) = %v, want %v", tt.model, tt.runner, got, tt.valid)
		}
	}
}

func TestWarnModelRunnerCompat_NoWarningForMatchingModel(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Variants: []Variant{{Name: "ctrl", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	// Should not panic or error — just logs warnings.
	warnModelRunnerCompat(s)
}

func TestWarnModelRunnerCompat_WarnsForMismatch(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Variants: []Variant{{Name: "ctrl", Runner: "claude-code", Model: "gpt-4o"}},
			},
		},
	}
	// Should not panic — warning is logged via log.Printf.
	warnModelRunnerCompat(s)
}

func TestValidate_RetryConfigValid(t *testing.T) {
	attempts := 3
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Retry:    &Retry{MaxAttempts: &attempts, Backoff: "exponential", Delay: "500ms", On: "all"},
				Verify:   []VerifyStep{{Type: "agent_exits_ok"}},
				Variants: []Variant{{Name: "ctrl", Runner: "claude-code", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	if err := validate(s); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_RetryMaxAttemptsInvalid(t *testing.T) {
	attempts := 0
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Retry:    &Retry{MaxAttempts: &attempts},
				Variants: []Variant{{Name: "c", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, "max_attempts must be >= 1")
}

func TestValidate_RetryInvalidBackoff(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Retry:    &Retry{Backoff: "linear"},
				Variants: []Variant{{Name: "c", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, `backoff must be 'fixed' or 'exponential'`)
}

func TestValidate_RetryInvalidDelay(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Retry:    &Retry{Delay: "not-a-duration"},
				Variants: []Variant{{Name: "c", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, `delay "not-a-duration" is not a valid duration`)
}

func TestValidate_RetryInvalidOn(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Retry:    &Retry{On: "never"},
				Variants: []Variant{{Name: "c", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, `on must be 'transient' or 'all'`)
}

func TestValidate_RetryOnDefaultsLevel(t *testing.T) {
	attempts := 0
	s := &Suite{
		Version:  1,
		Defaults: Defaults{Retry: &Retry{MaxAttempts: &attempts}},
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Variants: []Variant{{Name: "c", Model: "claude-sonnet-4-6"}},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, "defaults.retry: max_attempts must be >= 1")
}

func TestValidate_RetryOnVariantLevel(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Variants: []Variant{
					{Name: "c", Retry: &Retry{Backoff: "bad"}, Model: "claude-sonnet-4-6"},
				},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, "variant[0].retry: backoff must be")
}

func assertValidationContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q, got nil", substr)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	for _, e := range ve.Errors {
		if contains(e, substr) {
			return
		}
	}
	t.Errorf("expected error containing %q, got: %v", substr, ve.Errors)
}
