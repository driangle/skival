package suite

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"
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

// validateMatrixExclusive checks that no eval defines both matrix and treatments.
// This runs before matrix expansion.
func validateMatrixExclusive(s *Suite) error {
	var errs []string
	for i, eval := range s.Evals {
		if eval.Matrix == nil || len(eval.Matrix.Dimensions) == 0 {
			continue
		}
		hasTreatments := eval.Treatments.Control.Name != "" || len(eval.Treatments.Variations) > 0
		if hasTreatments {
			errs = append(errs, fmt.Sprintf("eval[%d]: cannot define both matrix and treatments", i))
		}
	}
	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
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

		// Prompt is required at eval level unless every treatment provides its own.
		if eval.Prompt == "" {
			if eval.Treatments.Control.Prompt == "" {
				errs = append(errs, fmt.Sprintf("%s: prompt is required (set on eval or on every treatment)", prefix))
			}
			for j, v := range eval.Treatments.Variations {
				if v.Prompt == "" {
					errs = append(errs, fmt.Sprintf("%s: variation[%d] %q has no prompt (set prompt on the eval or treatment)", prefix, j, v.Name))
				}
			}
		}

		if eval.Correctness.Compiles == "true" || eval.Correctness.Compiles == "false" {
			errs = append(errs, fmt.Sprintf("%s: compiles must be a build command string (e.g. \"go build ./...\"), not a boolean", prefix))
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

		// Skill and skills are mutually exclusive on each treatment.
		if eval.Treatments.Control.Skill != "" && len(eval.Treatments.Control.Skills) > 0 {
			errs = append(errs, fmt.Sprintf("%s: control treatment %q: cannot set both skill and skills", prefix, eval.Treatments.Control.Name))
		}
		for j, v := range eval.Treatments.Variations {
			if v.Skill != "" && len(v.Skills) > 0 {
				errs = append(errs, fmt.Sprintf("%s: variation[%d] %q: cannot set both skill and skills", prefix, j, v.Name))
			}
		}

		// Validate config_dir paths exist.
		if eval.Treatments.Control.ConfigDir != "" {
			if _, err := os.Stat(eval.Treatments.Control.ConfigDir); err != nil {
				errs = append(errs, fmt.Sprintf("%s: control treatment %q: config_dir %q does not exist", prefix, eval.Treatments.Control.Name, eval.Treatments.Control.ConfigDir))
			}
		}
		for j, v := range eval.Treatments.Variations {
			if v.ConfigDir != "" {
				if _, err := os.Stat(v.ConfigDir); err != nil {
					errs = append(errs, fmt.Sprintf("%s: variation[%d] %q: config_dir %q does not exist", prefix, j, v.Name, v.ConfigDir))
				}
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

	// Validate ranking weights.
	errs = append(errs, validateRankingWeights(s.Ranking)...)

	// Validate retry config at all levels.
	errs = append(errs, validateRetryConfig(s.Defaults.Retry, "defaults.retry")...)
	for i, eval := range s.Evals {
		prefix := fmt.Sprintf("eval[%d]", i)
		errs = append(errs, validateRetryConfig(eval.Retry, prefix+".retry")...)
		errs = append(errs, validateRetryConfig(eval.Treatments.Control.Retry, prefix+".control.retry")...)
		for j, v := range eval.Treatments.Variations {
			errs = append(errs, validateRetryConfig(v.Retry, fmt.Sprintf("%s.variation[%d].retry", prefix, j))...)
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

func validateRankingWeights(r *Ranking) []string {
	if r == nil {
		return nil
	}
	var errs []string
	w := r.Weights
	if w.Correctness < 0 {
		errs = append(errs, fmt.Sprintf("ranking.weights.correctness must be >= 0, got %g", w.Correctness))
	}
	if w.Cost < 0 {
		errs = append(errs, fmt.Sprintf("ranking.weights.cost must be >= 0, got %g", w.Cost))
	}
	if w.Duration < 0 {
		errs = append(errs, fmt.Sprintf("ranking.weights.duration must be >= 0, got %g", w.Duration))
	}
	sum := w.Correctness + w.Cost + w.Duration
	if math.Abs(sum-1.0) > 1e-9 {
		errs = append(errs, fmt.Sprintf("ranking.weights must sum to 1.0, got %g", sum))
	}
	return errs
}

var validBackoffs = map[string]bool{
	"":             true,
	"fixed":        true,
	"exponential":  true,
}

var validRetryOn = map[string]bool{
	"":          true,
	"transient": true,
	"all":       true,
}

func validateRetryConfig(r *Retry, path string) []string {
	if r == nil {
		return nil
	}
	var errs []string
	if r.MaxAttempts != nil && *r.MaxAttempts < 1 {
		errs = append(errs, fmt.Sprintf("%s: max_attempts must be >= 1, got %d", path, *r.MaxAttempts))
	}
	if !validBackoffs[r.Backoff] {
		errs = append(errs, fmt.Sprintf("%s: backoff must be 'fixed' or 'exponential', got %q", path, r.Backoff))
	}
	if r.Delay != "" {
		if _, err := time.ParseDuration(r.Delay); err != nil {
			errs = append(errs, fmt.Sprintf("%s: delay %q is not a valid duration", path, r.Delay))
		}
	}
	if !validRetryOn[r.On] {
		errs = append(errs, fmt.Sprintf("%s: on must be 'transient' or 'all', got %q", path, r.On))
	}
	return errs
}

// warnModelRunnerCompat logs warnings when a model ID doesn't look valid
// for the selected runner. This is advisory only and does not block loading.
func warnModelRunnerCompat(s *Suite) {
	for _, eval := range s.Evals {
		warnTreatmentModelRunner(eval.Treatments.Control, eval.Model)
		for _, v := range eval.Treatments.Variations {
			warnTreatmentModelRunner(v, eval.Model)
		}
	}
}

func warnTreatmentModelRunner(t Treatment, evalModel string) {
	model := t.Model
	if model == "" {
		model = evalModel
	}
	if model == "" {
		return
	}

	runner := t.Runner
	if runner == "" {
		runner = "claude-code"
	}

	if !modelLooksValidForRunner(model, runner) {
		log.Printf("WARNING: treatment %q: model %q may not be compatible with runner %q", t.Name, model, runner)
	}
}

// modelLooksValidForRunner checks if a model ID matches known patterns for a runner.
// Returns true if the model looks valid or if no patterns are known for the runner.
func modelLooksValidForRunner(model, runner string) bool {
	switch runner {
	case "claude-code":
		return strings.HasPrefix(model, "claude-")
	case "codex":
		return strings.HasPrefix(model, "gpt-") ||
			strings.HasPrefix(model, "o1-") ||
			strings.HasPrefix(model, "o3-") ||
			strings.HasPrefix(model, "codex-")
	default:
		return true
	}
}
