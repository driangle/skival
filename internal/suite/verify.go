package suite

import "fmt"

var validVerifyTypes = map[string]bool{
	"agent_exits_ok":  true,
	"check":           true,
	"check_output":    true,
	"output_contains": true,
	"command":         true,
	"file_contains":   true,
	"http_check":      true,
	"tcp_check":       true,
	"judge":           true,
}

// verifyTypeFields lists the type-specific YAML fields each verify type accepts.
// The common `type` and `name` fields are always allowed and omitted here.
// Setting a field not listed for a step's type is a validation error.
var verifyTypeFields = map[string]map[string]bool{
	"agent_exits_ok":  {},
	"check":           {"run": true},
	"check_output":    {"run": true},
	"output_contains": {"values": true},
	"command":         {"run": true, "exits": true, "stdout_contains": true},
	"file_contains":   {"path": true, "contains": true, "exists": true},
	"http_check":      {"url": true, "method": true, "status": true, "body_contains": true},
	"tcp_check":       {"host": true, "port": true},
	"judge":           {"criteria": true, "model": true},
}

// setVerifyFields returns the YAML names of the type-specific fields that are
// populated on the step. It mirrors the non-zero checks used by the
// required-field validation above.
func setVerifyFields(step VerifyStep) []string {
	checks := []struct {
		name string
		set  bool
	}{
		{"run", step.Run != ""},
		{"exits", step.Exits != nil},
		{"stdout_contains", step.StdoutContains != ""},
		{"values", len(step.Values) > 0},
		{"path", step.Path != ""},
		{"contains", step.Contains != ""},
		{"exists", step.Exists != nil},
		{"url", step.URL != ""},
		{"method", step.Method != ""},
		{"status", step.Status != nil},
		{"body_contains", step.BodyContains != ""},
		{"host", step.Host != ""},
		{"port", step.Port != 0},
		{"criteria", len(step.Criteria) > 0},
		{"model", step.Model != ""},
	}
	var set []string
	for _, c := range checks {
		if c.set {
			set = append(set, c.name)
		}
	}
	return set
}

func validateVerifySteps(steps []VerifyStep, prefix string) []string {
	var errs []string
	for i, step := range steps {
		sp := fmt.Sprintf("%s.verify[%d]", prefix, i)

		if step.Type == "" {
			errs = append(errs, fmt.Sprintf("%s: type is required", sp))
			continue
		}
		if !validVerifyTypes[step.Type] {
			errs = append(errs, fmt.Sprintf("%s: unknown type %q", sp, step.Type))
			continue
		}

		errs = append(errs, validateVerifyStepRequired(step, sp)...)
		errs = append(errs, validateVerifyStepFields(step, sp)...)
	}
	return errs
}

// validateVerifyStepRequired checks that a step of a known type has the fields
// its type requires.
func validateVerifyStepRequired(step VerifyStep, sp string) []string {
	var errs []string
	req := func(missing bool, msg string) {
		if missing {
			errs = append(errs, fmt.Sprintf("%s: %s", sp, msg))
		}
	}
	switch step.Type {
	case "check":
		req(step.Run == "", "check requires run")
		req(step.Run == "true" || step.Run == "false", "check run must be a shell command, not a boolean")
	case "check_output":
		req(step.Run == "", "check_output requires run")
	case "output_contains":
		req(len(step.Values) == 0, "output_contains requires values")
	case "command":
		req(step.Run == "", "command requires run")
	case "file_contains":
		req(step.Path == "", "file_contains requires path")
	case "http_check":
		req(step.URL == "", "http_check requires url")
	case "tcp_check":
		req(step.Host == "", "tcp_check requires host")
		req(step.Port == 0, "tcp_check requires port")
	case "judge":
		req(len(step.Criteria) == 0, "judge requires criteria")
	}
	return errs
}

// validateVerifyStepFields rejects fields that don't belong to the step's type.
func validateVerifyStepFields(step VerifyStep, sp string) []string {
	var errs []string
	allowed := verifyTypeFields[step.Type]
	for _, f := range setVerifyFields(step) {
		if !allowed[f] {
			errs = append(errs, fmt.Sprintf("%s: field %q is not valid for type %q", sp, f, step.Type))
		}
	}
	return errs
}
