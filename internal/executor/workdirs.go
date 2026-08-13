package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/driangle/skival/internal/result"
)

// isolatedWorkdirPrefix is the base-name prefix of directories created by
// createIsolatedDir. Only directories matching it are ever removed, so a
// user-provided eval/variant Dir is never touched.
const isolatedWorkdirPrefix = "skival-isolate-"

// ValidateKeepWorkdirs reports whether v is an accepted --keep-workdirs value.
// The empty string is allowed and resolves to the "failed" default.
func ValidateKeepWorkdirs(v string) error {
	switch v {
	case "", "all", "failed", "none":
		return nil
	default:
		return fmt.Errorf("invalid --keep-workdirs %q: must be one of all, failed, none", v)
	}
}

// cleanupWorkdirs removes isolated sample workdirs that the policy says to drop,
// clearing their WorkDir so the report only references retained directories.
func cleanupWorkdirs(sr *result.SuiteResult, policy string) {
	if sr == nil || policy == "all" {
		return
	}
	for ei := range sr.Evals {
		variants := sr.Evals[ei].Variants
		for vi := range variants {
			runs := variants[vi].Runs
			for ri := range runs {
				dropWorkdir(&runs[ri], policy)
			}
		}
	}
}

// dropWorkdir removes and clears the run's WorkDir when the policy and the
// isolation guard both allow it.
func dropWorkdir(run *result.RunResult, policy string) {
	if !shouldDropWorkdir(run, policy) || !isIsolatedWorkdir(run.WorkDir) {
		return
	}
	_ = os.RemoveAll(run.WorkDir)
	run.WorkDir = ""
}

// shouldDropWorkdir decides, by policy, whether a run's workdir should be
// removed. "none" drops everything; "failed"/empty drops only passing samples,
// keeping failing and unverified ones.
func shouldDropWorkdir(run *result.RunResult, policy string) bool {
	switch policy {
	case "none":
		return true
	case "failed", "":
		return run.Pass != nil && *run.Pass
	default:
		return false
	}
}

// isIsolatedWorkdir reports whether dir is a directory created by
// createIsolatedDir, guarding against deleting user-provided directories.
func isIsolatedWorkdir(dir string) bool {
	if dir == "" {
		return false
	}
	return strings.HasPrefix(filepath.Base(dir), isolatedWorkdirPrefix)
}
