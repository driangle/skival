package suite

import (
	"fmt"

	execrunner "github.com/driangle/skival/internal/runners/exec"
)

// validateExecVariant validates the runner_config of an exec-runner variant.
// It reports a missing/invalid command and an out-of-range prompt_via. Prefixed
// errors are appended to the caller's list.
func validateExecVariant(v Variant, vp string) []string {
	cfg, err := execrunner.ParseConfig(v.RunnerConfig)
	if err != nil {
		return []string{fmt.Sprintf("%s %q: invalid exec runner_config: %v", vp, v.Name, err)}
	}
	var errs []string
	for _, msg := range cfg.Validate() {
		errs = append(errs, fmt.Sprintf("%s %q: %s", vp, v.Name, msg))
	}
	return errs
}
