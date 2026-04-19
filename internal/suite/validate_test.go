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
				Variants: []Treatment{
					{Name: "baseline", Runner: "claude-code"},
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
			{ID: "e1", Prompt: "p", Variants: []Treatment{{Name: "c"}}},
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
		Evals:   []Eval{{Prompt: "p", Variants: []Treatment{{Name: "c"}}}},
	}

	err := validate(s)
	assertValidationContains(t, err, "id is required")
}

func TestValidate_EvalPromptRequired(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals:   []Eval{{ID: "e1", Variants: []Treatment{{Name: "c"}}}},
	}

	err := validate(s)
	assertValidationContains(t, err, "has no prompt")
}

func TestValidate_TreatmentPromptOverridesEvalPrompt(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:    "e1",
				Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", Prompt: "do A", Runner: "claude-code"},
					{Name: "v1", Prompt: "do B", Runner: "claude-code"},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("treatment-level prompts should satisfy prompt requirement, got: %v", err)
	}
}

func TestValidate_VariantMissingPromptWhenNoEvalPrompt(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:    "e1",
				Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", Prompt: "do A"},
					{Name: "v1"},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `variant[1] "v1" has no prompt`)
}

func TestValidate_ConfigDirMustExist(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", ConfigDir: "/nonexistent/config/dir"},
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
				Variants: []Treatment{
					{Name: "ctrl", ConfigDir: dir, Runner: "claude-code"},
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
			{ID: "dup", Prompt: "p1", Variants: []Treatment{{Name: "c"}}},
			{ID: "dup", Prompt: "p2", Variants: []Treatment{{Name: "c"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `duplicate id "dup"`)
}

func TestValidate_AtLeastOneVariantRequired(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals:   []Eval{{ID: "e1", Prompt: "p"}},
	}

	err := validate(s)
	assertValidationContains(t, err, "at least one variant is required")
}

func TestValidate_VariantNameRequired(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Variants: []Treatment{{}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, "name is required")
}

func TestValidate_MultipleErrors(t *testing.T) {
	s := &Suite{
		Version: 0,
		Evals:   []Eval{{}},
	}

	var ve *ValidationError
	err := validate(s)
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got: %v", err)
	}

	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 errors (version, id, variants), got %d: %v",
			len(ve.Errors), ve.Errors)
	}
}

func TestValidate_ModelRequiredOnVariant(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Variants: []Treatment{{Name: "ctrl"}}},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `"ctrl" has no model`)
}

func TestValidate_ModelRequiredOnSecondVariant(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Variants: []Treatment{
					{Name: "ctrl", Model: "claude-sonnet-4-6"},
					{Name: "v1"},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `"v1" has no model`)
}

func TestValidate_EvalModelCoversAllVariants(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", Runner: "claude-code"},
					{Name: "v1", Runner: "claude-code"},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("eval-level model should cover all variants, got: %v", err)
	}
}

func TestValidate_VariantLevelModelsWithoutEvalModel(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p",
				Variants: []Treatment{
					{Name: "ctrl", Model: "claude-sonnet-4-6", Runner: "claude-code"},
					{Name: "v1", Model: "claude-opus-4-6", Runner: "claude-code"},
				},
			},
		},
	}

	if err := validate(s); err != nil {
		t.Fatalf("per-variant models should be valid, got: %v", err)
	}
}

func TestValidate_ValidRunners(t *testing.T) {
	for _, runner := range []string{"claude-code", "ollama", "codex", "aider"} {
		s := &Suite{
			Version: 1,
			Evals: []Eval{
				{
					ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
					Runner:   runner,
					Variants: []Treatment{{Name: "ctrl", Runner: runner}},
				},
			},
		}
		if err := validate(s); err != nil {
			t.Errorf("runner %q should be valid, got: %v", runner, err)
		}
	}
}

func TestValidate_MissingRunnerOnVariant(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{{Name: "ctrl"}},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `"ctrl" has no runner`)
}

func TestValidate_MissingRunnerOnSecondVariant(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", Runner: "claude-code"},
					{Name: "v1", Model: "claude-sonnet-4-6"},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `"v1" has no runner`)
}

func TestValidate_UnknownRunnerOnDefaults(t *testing.T) {
	s := &Suite{
		Version:  1,
		Defaults: Defaults{Runner: "unknown"},
		Evals: []Eval{
			{ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6", Variants: []Treatment{{Name: "c"}}},
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
				Runner:   "bad-runner",
				Variants: []Treatment{{Name: "c"}},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `eval[0]: unknown runner "bad-runner"`)
}

func TestValidate_UnknownRunnerOnVariant(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{{Name: "ctrl", Runner: "nope"}},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `"ctrl": unknown runner "nope"`)
}

func TestValidate_SkillAndSkillsMutuallyExclusiveOnVariant(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", Skill: "a.md", Skills: []string{"b.md"}},
				},
			},
		},
	}

	err := validate(s)
	assertValidationContains(t, err, `"ctrl": cannot set both skill and skills`)
}

func TestValidate_SkillsAloneIsValid(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "ctrl", Skills: []string{"a.md", "b.md"}, Runner: "claude-code"},
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
				Variants: []Treatment{{Name: "ctrl"}},
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
				Variants: []Treatment{{Name: "ctrl", Runner: "claude-code"}},
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
				Retry:    &Retry{MaxAttempts: &attempts, Backoff: "exponential", Delay: "500ms", On: "all"},
				Variants: []Treatment{{Name: "ctrl", Runner: "claude-code"}},
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
				Retry:    &Retry{MaxAttempts: &attempts},
				Variants: []Treatment{{Name: "c"}},
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
				Retry:    &Retry{Backoff: "linear"},
				Variants: []Treatment{{Name: "c"}},
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
				Retry:    &Retry{Delay: "not-a-duration"},
				Variants: []Treatment{{Name: "c"}},
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
				Retry:    &Retry{On: "never"},
				Variants: []Treatment{{Name: "c"}},
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
				Variants: []Treatment{{Name: "c"}},
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
				ID: "e1", Prompt: "p", Model: "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "c", Retry: &Retry{Backoff: "bad"}},
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

func TestValidate_VerifyStepMissingType(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{}}
	})
	err := validate(s)
	requireValidationError(t, err, "type is required")
}

func TestValidate_VerifyStepUnknownType(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "unknown"}}
	})
	err := validate(s)
	requireValidationError(t, err, `unknown type "unknown"`)
}

func TestValidate_HttpCheckMissingURL(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "http_check"}}
	})
	err := validate(s)
	requireValidationError(t, err, "http_check requires url")
}

func TestValidate_FileContainsMissingPath(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "file_contains"}}
	})
	err := validate(s)
	requireValidationError(t, err, "file_contains requires path")
}

func TestValidate_CommandMissingRun(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "command"}}
	})
	err := validate(s)
	requireValidationError(t, err, "command requires run")
}

func TestValidate_TcpCheckMissingHostAndPort(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "tcp_check"}}
	})
	err := validate(s)
	ve := &ValidationError{}
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	requireValidationError(t, err, "tcp_check requires host")
	requireValidationError(t, err, "tcp_check requires port")
}

func TestValidate_CheckBooleanRejected(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "check", Run: "true"}}
	})
	err := validate(s)
	requireValidationError(t, err, "check run must be a shell command, not a boolean")
}

func TestValidate_JudgeMissingCriteria(t *testing.T) {
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{{Type: "judge"}}
	})
	err := validate(s)
	requireValidationError(t, err, "judge requires criteria")
}

func TestValidate_ValidVerifySteps(t *testing.T) {
	trueVal := true
	exitZero := 0
	status200 := 200
	s := validSuiteWith(func(e *Eval) {
		e.Verify = []VerifyStep{
			{Type: "agent_exits_ok"},
			{Type: "check", Run: "go build ./..."},
			{Type: "check_output", Run: "exit 0"},
			{Type: "output_contains", Values: []string{"ok"}},
			{Type: "http_check", URL: "http://localhost", Status: &status200},
			{Type: "file_contains", Path: "/tmp/test", Exists: &trueVal},
			{Type: "command", Run: "echo hi", Exits: &exitZero},
			{Type: "tcp_check", Host: "localhost", Port: 8080},
			{Type: "judge", Criteria: []string{"is correct"}},
		}
	})
	if err := validate(s); err != nil {
		t.Fatalf("expected valid verify steps to pass validation: %v", err)
	}
}

func validSuiteWith(modify func(*Eval)) *Suite {
	s := &Suite{
		Version: 1,
		Evals: []Eval{
			{
				ID:     "eval-1",
				Prompt: "do something",
				Model:  "claude-sonnet-4-6",
				Variants: []Treatment{
					{Name: "baseline", Runner: "claude-code"},
				},
			},
		},
	}
	modify(&s.Evals[0])
	return s
}

func requireValidationError(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q, got nil", substr)
	}
	if !contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got: %s", substr, err.Error())
	}
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
