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

		if eval.Treatments.Control.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: control treatment name is required", prefix))
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}
