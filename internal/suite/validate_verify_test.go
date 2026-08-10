package suite

import (
	"errors"
	"testing"
)

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

func TestValidate_VerifyStepWrongFieldForType(t *testing.T) {
	cases := []struct {
		name string
		step VerifyStep
		want string
	}{
		{
			name: "port on judge",
			step: VerifyStep{Type: "judge", Criteria: []string{"is correct"}, Port: 8080},
			want: `field "port" is not valid for type "judge"`,
		},
		{
			name: "url on tcp_check",
			step: VerifyStep{Type: "tcp_check", Host: "localhost", Port: 8080, URL: "http://x"},
			want: `field "url" is not valid for type "tcp_check"`,
		},
		{
			name: "criteria on check",
			step: VerifyStep{Type: "check", Run: "go build ./...", Criteria: []string{"nope"}},
			want: `field "criteria" is not valid for type "check"`,
		},
		{
			name: "values on http_check",
			step: VerifyStep{Type: "http_check", URL: "http://localhost", Values: []string{"x"}},
			want: `field "values" is not valid for type "http_check"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validSuiteWith(func(e *Eval) {
				e.Verify = []VerifyStep{tc.step}
			})
			err := validate(s)
			requireValidationError(t, err, tc.want)
		})
	}
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
