package suite

import (
	"fmt"
	"strings"
)

// ValidationError collects multiple validation problems.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed with %d error(s):\n- %s",
		len(e.Errors), strings.Join(e.Errors, "\n- "))
}

var validComplexities = map[string]bool{
	"":       true,
	"low":    true,
	"medium": true,
	"high":   true,
}

var validRunners = map[string]bool{
	"":            true,
	"claude-code": true,
	"ollama":      true,
	"codex":       true,
	"aider":       true,
}

// validate checks the suite for structural problems and returns a
// *ValidationError if any are found.
func validate(s *Suite) error {
	var errs []string

	if s.Version <= 0 {
		errs = append(errs, "version must be greater than 0")
	}

	if len(s.Evals) == 0 {
		errs = append(errs, "at least one eval is required")
	}

	if !validRunners[s.Defaults.Runner] {
		errs = append(errs, fmt.Sprintf("defaults: unknown runner %q", s.Defaults.Runner))
	}

	seenIDs := make(map[string]bool)
	for i, eval := range s.Evals {
		prefix := fmt.Sprintf("eval[%d]", i)

		if eval.ID == "" {
			errs = append(errs, fmt.Sprintf("%s: id is required", prefix))
		} else if seenIDs[eval.ID] {
			errs = append(errs, fmt.Sprintf("%s: duplicate id %q", prefix, eval.ID))
		} else {
			seenIDs[eval.ID] = true
		}

		if eval.Prompt == "" {
			errs = append(errs, fmt.Sprintf("%s: prompt is required", prefix))
		}

		if !validComplexities[eval.Complexity] {
			errs = append(errs, fmt.Sprintf("%s: invalid complexity %q (must be low, medium, or high)", prefix, eval.Complexity))
		}

		if !validRunners[eval.Runner] {
			errs = append(errs, fmt.Sprintf("%s: unknown runner %q", prefix, eval.Runner))
		}

		if eval.Treatments.Control.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: control treatment name is required", prefix))
		}

		if !validRunners[eval.Treatments.Control.Runner] {
			errs = append(errs, fmt.Sprintf("%s: control treatment %q: unknown runner %q", prefix, eval.Treatments.Control.Name, eval.Treatments.Control.Runner))
		}
		for j, v := range eval.Treatments.Variations {
			if !validRunners[v.Runner] {
				errs = append(errs, fmt.Sprintf("%s: variation[%d] %q: unknown runner %q", prefix, j, v.Name, v.Runner))
			}
		}

		// Every treatment must resolve to a model (treatment-level or eval-level).
		if eval.Model == "" {
			if eval.Treatments.Control.Model == "" {
				errs = append(errs, fmt.Sprintf("%s: control treatment %q has no model (set model on the eval or treatment)", prefix, eval.Treatments.Control.Name))
			}
			for j, v := range eval.Treatments.Variations {
				if v.Model == "" {
					errs = append(errs, fmt.Sprintf("%s: variation[%d] %q has no model (set model on the eval or treatment)", prefix, j, v.Name))
				}
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}
