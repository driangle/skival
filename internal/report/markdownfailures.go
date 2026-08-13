package report

import (
	"fmt"
	"io"

	"github.com/driangle/skival/internal/result"
)

// writeFailuresSection lists each failed run with the first failing step's name
// and message, so the report explains why a sample failed. It renders nothing
// when no runs failed.
func writeFailuresSection(w io.Writer, sr *result.SuiteResult) {
	if !hasFailedRun(sr) {
		return
	}

	fmt.Fprintf(w, "## Failures\n\n")
	for _, eval := range sr.Evals {
		name := evalDisplayName(eval)
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				writeFailureLine(w, name, v.Name, run)
			}
		}
	}
	fmt.Fprintln(w)
}

// hasFailedRun reports whether any run in the suite explicitly failed.
func hasFailedRun(sr *result.SuiteResult) bool {
	for _, eval := range sr.Evals {
		for _, v := range eval.Variants {
			for _, run := range v.Runs {
				if run.Pass != nil && !*run.Pass {
					return true
				}
			}
		}
	}
	return false
}

// writeFailureLine renders one failed run's bullet, naming the first failing
// step and its message. It writes nothing for passing or errored runs.
func writeFailureLine(w io.Writer, evalName, variant string, run result.RunResult) {
	if run.Pass == nil || *run.Pass {
		return
	}
	stepName, reason := "verification", "failed"
	if step, ok := firstFailingStep(run); ok {
		stepName = step.Name
		if step.Reason != "" {
			reason = step.Reason
		}
	}
	fmt.Fprintf(w, "- **%s** > %s > sample %d: %s — %s\n",
		evalName, variant, run.Sample, stepName, reason)
}

// firstFailingStep returns the first non-passing step of a run, if any.
func firstFailingStep(run result.RunResult) (result.StepResult, bool) {
	for _, s := range run.Steps {
		if !s.Pass {
			return s, true
		}
	}
	return result.StepResult{}, false
}
