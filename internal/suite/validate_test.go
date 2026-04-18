package suite

import (
	"errors"
	"testing"
)

func TestValidate_ValidSuite(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:     "eval-1",
				Prompt: "do something",
				Model:  "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "baseline"},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_VersionRequired(t *testing.T) {
	s := &Suite{
		Version: 0,
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Treatments: Treatments{Control: Treatment{Name: "c"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, "version must be greater than 0")
}

func TestValidate_AtLeastOneEval(t *testing.T) {
	s := &Suite{Version: 1, Evals: nil}

	err := validate(s)
	assertValidationContains(t, err, "at least one eval is required")
}

func TestValidate_EvalIDRequired(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals:   []Eval{{Prompt: "p", Treatments: Treatments{Control: Treatment{Name: "c"}}}},
	}

	err := validate(s)
	assertValidationContains(t, err, "id is required")
}

func TestValidate_EvalPromptRequired(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals:   []Eval{{ID: "e1", Treatments: Treatments{Control: Treatment{Name: "c"}}}},
	}

	err := validate(s)
	assertValidationContains(t, err, "prompt is required")
}

func TestValidate_TreatmentPromptOverridesEvalPrompt(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:    "e1",
				Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control:    Treatment{Name: "ctrl", Prompt: "do A"},
					Variations: []Treatment{{Name: "v1", Prompt: "do B"}},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("treatment-level prompts should satisfy prompt requirement, got: %v", err)
	}
}

func TestValidate_VariationMissingPromptWhenNoEvalPrompt(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:    "e1",
				Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control:    Treatment{Name: "ctrl", Prompt: "do A"},
					Variations: []Treatment{{Name: "v1"}},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `variation[0] "v1" has no prompt`)
}

func TestValidate_ConfigDirMustExist(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl", ConfigDir: "/nonexistent/config/dir"},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `config_dir "/nonexistent/config/dir" does not exist`)
}

func TestValidate_ConfigDirValid(t *testing.T) {
	dir := t.TempDir()
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl", ConfigDir: dir},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("valid config_dir should pass, got: %v", err)
	}
}

func TestValidate_DuplicateEvalIDs(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{ID: "dup", Prompt: "p1", Treatments: Treatments{Control: Treatment{Name: "c"}}},
			{ID: "dup", Prompt: "p2", Treatments: Treatments{Control: Treatment{Name: "c"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `duplicate id "dup"`)
}

func TestValidate_InvalidComplexity(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Complexity: "extreme", Treatments: Treatments{Control: Treatment{Name: "c"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `invalid complexity "extreme"`)
}

func TestValidate_ValidComplexities(t *testing.T) {
	for _, c := range []string{"", "low", "medium", "high"} {
		s := &Suite{
			Version: 1,
			Evals: []Eval{
				{ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6", Complexity: c, Treatments: Treatments{Control: Treatment{Name: "c"}}},
			},
		}
		if err := validate(s); err != nil {
			t.Errorf("complexity %q should be valid, got: %v", c, err)
		}
	}
}

func TestValidate_ControlTreatmentNameRequired(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals:   []Eval{{ID: "e1", Prompt: "p"}},
	}

	err := validate(s)
	assertValidationContains(t, err, "control treatment name is required")
}

func TestValidate_MultipleErrors(t *testing.T) {
	s := &Suite{
		Version: 0,
		Evals:   []Eval{{Complexity: "wrong"}},
	}

	var ve *ValidationError
	err := validate(s)
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got: %v", err)
	}

	if len(ve.Errors) < 4 {
		t.Errorf("expected at least 4 errors (version, id, prompt, complexity, control), got %d: %v",
			len(ve.Errors), ve.Errors)
	}
}

func TestValidate_ModelRequiredOnControl(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Treatments: Treatments{Control: Treatment{Name: "ctrl"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `control treatment "ctrl" has no model`)
}

func TestValidate_ModelRequiredOnVariation(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Treatments: Treatments{
					Control:    Treatment{Name: "ctrl", Model: "claude-sonnet-4-6"},
					Variations: []Treatment{{Name: "v1"}},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `variation[0] "v1" has no model`)
}

func TestValidate_EvalModelCoversAllTreatments(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control:    Treatment{Name: "ctrl"},
					Variations: []Treatment{{Name: "v1"}},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("eval-level model should cover all treatments, got: %v", err)
	}
}

func TestValidate_TreatmentLevelModelsWithoutEvalModel(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Treatments: Treatments{
					Control:    Treatment{Name: "ctrl", Model: "claude-sonnet-4-6"},
					Variations: []Treatment{{Name: "v1", Model: "claude-opus-4-6"}},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("per-treatment models should be valid, got: %v", err)
	}
}

func TestValidate_ValidRunners(t *testing.T) {
	for _, runner := range []string{"", "claude-code", "ollama"} {
		s := &Suite{
			Version: 1,
			Evals: []Eval{
				{
					ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
					Runner: runner,
					Treatments: Treatments{
						Control: Treatment{Name: "ctrl", Runner: runner},
					},
				},
			},
		}
		if err := validate(s); err != nil {
			t.Errorf("runner %q should be valid, got: %v", runner, err)
		}
	}
}

func TestValidate_UnknownRunnerOnDefaults(t *testing.T) {
	s := &Suite{
		Version:  1,
		Defaults: Defaults{Runner: "unknown"},
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6", Treatments: Treatments{Control: Treatment{Name: "c"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `defaults: unknown runner "unknown"`)
}

func TestValidate_UnknownRunnerOnEval(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Runner:     "bad-runner",
				Treatments: Treatments{Control: Treatment{Name: "c"}},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `eval[0]: unknown runner "bad-runner"`)
}

func TestValidate_UnknownRunnerOnControlTreatment(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl", Runner: "nope"},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `control treatment "ctrl": unknown runner "nope"`)
}

func TestValidate_UnknownRunnerOnVariation(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control:    Treatment{Name: "ctrl"},
					Variations: []Treatment{{Name: "v1", Model: "claude-sonnet-4-6", Runner: "fake"}},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `variation[0] "v1": unknown runner "fake"`)
}

func TestValidate_SkillAndSkillsMutuallyExclusiveOnControl(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl", Skill: "a.md", Skills: []string{"b.md"}},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `control treatment "ctrl": cannot set both skill and skills`)
}

func TestValidate_SkillAndSkillsMutuallyExclusiveOnVariation(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl"},
					Variations: []Treatment{
						{Name: "v1", Skill: "a.md", Skills: []string{"b.md"}},
					},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `variation[0] "v1": cannot set both skill and skills`)
}

func TestValidate_SkillsAloneIsValid(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl", Skills: []string{"a.md", "b.md"}},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("skills alone should be valid, got: %v", err)
	}
}

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
		{"any-model", "aider", true},           // aider accepts anything
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl"},
				},
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
				ID: "e1", Prompt: "p", Model: "gpt-4o",
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl", Runner: "claude-code"},
				},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Retry: &Retry{MaxAttempts: &attempts, Backoff: "exponential", Delay: "500ms", On: "all"},
				Treatments: Treatments{
					Control: Treatment{Name: "ctrl"},
				},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Retry:      &Retry{MaxAttempts: &attempts},
				Treatments: Treatments{Control: Treatment{Name: "c"}},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Retry:      &Retry{Backoff: "linear"},
				Treatments: Treatments{Control: Treatment{Name: "c"}},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Retry:      &Retry{Delay: "not-a-duration"},
				Treatments: Treatments{Control: Treatment{Name: "c"}},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Retry:      &Retry{On: "never"},
				Treatments: Treatments{Control: Treatment{Name: "c"}},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{Control: Treatment{Name: "c"}},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, "defaults.retry: max_attempts must be >= 1")
}

func TestValidate_RetryOnTreatmentLevel(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Treatments: Treatments{
					Control: Treatment{Name: "c", Retry: &Retry{Backoff: "bad"}},
				},
			},
		},
	}
	err := validate(s)
	assertValidationContains(t, err, "eval[0].control.retry: backoff must be")
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
