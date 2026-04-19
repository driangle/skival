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

var validRunners = map[string]bool{
	"claude-code": true,
	"ollama":      true,
	"codex":       true,
	"aider":       true,
}

// validateMatrixExclusive checks that no eval defines both matrix and variants.
// This runs before matrix expansion.
func validateMatrixExclusive(s *Suite) error {
	var errs []string
	for i, eval := range s.Evals {
		if eval.Matrix == nil || len(eval.Matrix.Dimensions) == 0 {
			continue
		}
		if len(eval.Variants) > 0 {
			errs = append(errs, fmt.Sprintf("eval[%d]: cannot define both matrix and variants", i))
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

	if s.Defaults.Runner != "" && !validRunners[s.Defaults.Runner] {
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

		// Prompt is required at eval level unless every variant provides its own.
		if eval.Prompt == "" {
			for j, v := range eval.Variants {
				if v.Prompt == "" {
					errs = append(errs, fmt.Sprintf("%s: variant[%d] %q has no prompt (set prompt on the eval or variant)", prefix, j, v.Name))
				}
			}
		}

		errs = append(errs, validateVerifySteps(eval.Verify, prefix)...)

		if eval.Runner != "" && !validRunners[eval.Runner] {
			errs = append(errs, fmt.Sprintf("%s: unknown runner %q", prefix, eval.Runner))
		}

		if len(eval.Variants) == 0 {
			errs = append(errs, fmt.Sprintf("%s: at least one variant is required", prefix))
		}

		for j, v := range eval.Variants {
			vp := fmt.Sprintf("%s.variant[%d]", prefix, j)

			if v.Name == "" {
				errs = append(errs, fmt.Sprintf("%s: name is required", vp))
			}

			// Every variant must resolve to a runner.
			if v.Runner == "" {
				errs = append(errs, fmt.Sprintf("%s %q has no runner (set runner on the eval, defaults, or variant)", vp, v.Name))
			} else if !validRunners[v.Runner] {
				errs = append(errs, fmt.Sprintf("%s %q: unknown runner %q", vp, v.Name, v.Runner))
			}

			// Skill and skills are mutually exclusive.
			if v.Skill != "" && len(v.Skills) > 0 {
				errs = append(errs, fmt.Sprintf("%s %q: cannot set both skill and skills", vp, v.Name))
			}

			// Validate config_dir paths exist.
			if v.ConfigDir != "" {
				if _, err := os.Stat(v.ConfigDir); err != nil {
					errs = append(errs, fmt.Sprintf("%s %q: config_dir %q does not exist", vp, v.Name, v.ConfigDir))
				}
			}

			// Every variant must resolve to a model.
			if v.Model == "" {
				errs = append(errs, fmt.Sprintf("%s %q has no model (set model on the eval, defaults, or variant)", vp, v.Name))
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
		for j, v := range eval.Variants {
			errs = append(errs, validateRetryConfig(v.Retry, fmt.Sprintf("%s.variant[%d].retry", prefix, j))...)
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
		for _, v := range eval.Variants {
			warnVariantModelRunner(v)
		}
	}
}

func warnVariantModelRunner(v Variant) {
	if v.Model == "" || v.Runner == "" {
		return
	}

	if !modelLooksValidForRunner(v.Model, v.Runner) {
		log.Printf("WARNING: variant %q: model %q may not be compatible with runner %q", v.Name, v.Model, v.Runner)
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

		switch step.Type {
		case "check":
			if step.Run == "" {
				errs = append(errs, fmt.Sprintf("%s: check requires run", sp))
			}
			if step.Run == "true" || step.Run == "false" {
				errs = append(errs, fmt.Sprintf("%s: check run must be a shell command, not a boolean", sp))
			}
		case "check_output":
			if step.Run == "" {
				errs = append(errs, fmt.Sprintf("%s: check_output requires run", sp))
			}
		case "output_contains":
			if len(step.Values) == 0 {
				errs = append(errs, fmt.Sprintf("%s: output_contains requires values", sp))
			}
		case "command":
			if step.Run == "" {
				errs = append(errs, fmt.Sprintf("%s: command requires run", sp))
			}
		case "file_contains":
			if step.Path == "" {
				errs = append(errs, fmt.Sprintf("%s: file_contains requires path", sp))
			}
		case "http_check":
			if step.URL == "" {
				errs = append(errs, fmt.Sprintf("%s: http_check requires url", sp))
			}
		case "tcp_check":
			if step.Host == "" {
				errs = append(errs, fmt.Sprintf("%s: tcp_check requires host", sp))
			}
			if step.Port == 0 {
				errs = append(errs, fmt.Sprintf("%s: tcp_check requires port", sp))
			}
		case "judge":
			if len(step.Criteria) == 0 {
				errs = append(errs, fmt.Sprintf("%s: judge requires criteria", sp))
			}
		}
	}
	return errs
}

