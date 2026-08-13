package executor

import (
	"os"
	"testing"

	"github.com/driangle/skival/internal/result"
)

// mkIsolatedDir creates a real temp dir with the skival-isolate- prefix so the
// cleanup guard treats it as an isolated workdir.
func mkIsolatedDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "skival-isolate-*")
	if err != nil {
		t.Fatalf("creating isolated dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func boolp(b bool) *bool { return &b }

// suiteWithRuns builds a SuiteResult with a single eval/variant whose runs are
// the provided runs.
func suiteWithRuns(runs ...result.RunResult) *result.SuiteResult {
	return &result.SuiteResult{
		Evals: []result.EvalResult{
			{Variants: []result.VariantResult{{Name: "v1", Runs: runs}}},
		},
	}
}

func dirExists(dir string) bool {
	_, err := os.Stat(dir)
	return err == nil
}

func runsOf(sr *result.SuiteResult) []result.RunResult {
	return sr.Evals[0].Variants[0].Runs
}

func TestCleanupWorkdirs_FailedPolicy(t *testing.T) {
	passDir := mkIsolatedDir(t)
	failDir := mkIsolatedDir(t)
	unverifiedDir := mkIsolatedDir(t)

	sr := suiteWithRuns(
		result.RunResult{Sample: 1, Pass: boolp(true), WorkDir: passDir},
		result.RunResult{Sample: 2, Pass: boolp(false), WorkDir: failDir},
		result.RunResult{Sample: 3, Pass: nil, WorkDir: unverifiedDir},
	)

	cleanupWorkdirs(sr, "failed")

	runs := runsOf(sr)
	if dirExists(passDir) {
		t.Errorf("passing sample dir %q should have been removed", passDir)
	}
	if runs[0].WorkDir != "" {
		t.Errorf("passing sample WorkDir should be cleared, got %q", runs[0].WorkDir)
	}
	if !dirExists(failDir) {
		t.Errorf("failing sample dir %q should be preserved", failDir)
	}
	if runs[1].WorkDir != failDir {
		t.Errorf("failing sample WorkDir should be preserved, got %q", runs[1].WorkDir)
	}
	if !dirExists(unverifiedDir) {
		t.Errorf("unverified sample dir %q should be preserved", unverifiedDir)
	}
	if runs[2].WorkDir != unverifiedDir {
		t.Errorf("unverified sample WorkDir should be preserved, got %q", runs[2].WorkDir)
	}
}

func TestCleanupWorkdirs_EmptyPolicyDefaultsToFailed(t *testing.T) {
	passDir := mkIsolatedDir(t)
	sr := suiteWithRuns(result.RunResult{Sample: 1, Pass: boolp(true), WorkDir: passDir})

	cleanupWorkdirs(sr, "")

	if dirExists(passDir) {
		t.Errorf("empty policy should behave like failed and remove passing dir %q", passDir)
	}
}

func TestCleanupWorkdirs_AllPolicyKeepsEverything(t *testing.T) {
	passDir := mkIsolatedDir(t)
	failDir := mkIsolatedDir(t)
	sr := suiteWithRuns(
		result.RunResult{Sample: 1, Pass: boolp(true), WorkDir: passDir},
		result.RunResult{Sample: 2, Pass: boolp(false), WorkDir: failDir},
	)

	cleanupWorkdirs(sr, "all")

	runs := runsOf(sr)
	if !dirExists(passDir) || !dirExists(failDir) {
		t.Error("all policy should preserve every dir")
	}
	if runs[0].WorkDir != passDir || runs[1].WorkDir != failDir {
		t.Error("all policy should leave WorkDir untouched")
	}
}

func TestCleanupWorkdirs_NonePolicyRemovesAll(t *testing.T) {
	passDir := mkIsolatedDir(t)
	failDir := mkIsolatedDir(t)
	unverifiedDir := mkIsolatedDir(t)
	sr := suiteWithRuns(
		result.RunResult{Sample: 1, Pass: boolp(true), WorkDir: passDir},
		result.RunResult{Sample: 2, Pass: boolp(false), WorkDir: failDir},
		result.RunResult{Sample: 3, Pass: nil, WorkDir: unverifiedDir},
	)

	cleanupWorkdirs(sr, "none")

	for _, dir := range []string{passDir, failDir, unverifiedDir} {
		if dirExists(dir) {
			t.Errorf("none policy should remove dir %q", dir)
		}
	}
	for i, run := range runsOf(sr) {
		if run.WorkDir != "" {
			t.Errorf("none policy should clear WorkDir of run %d, got %q", i, run.WorkDir)
		}
	}
}

func TestCleanupWorkdirs_GuardsUserDir(t *testing.T) {
	userDir, err := os.MkdirTemp("", "user-dir-*")
	if err != nil {
		t.Fatalf("creating user dir: %v", err)
	}
	defer os.RemoveAll(userDir)

	sr := suiteWithRuns(result.RunResult{Sample: 1, Pass: boolp(true), WorkDir: userDir})

	cleanupWorkdirs(sr, "none")

	if !dirExists(userDir) {
		t.Errorf("non-isolated user dir %q must never be removed", userDir)
	}
	if runsOf(sr)[0].WorkDir != userDir {
		t.Error("non-isolated user dir WorkDir must be preserved")
	}
}

func TestValidateKeepWorkdirs(t *testing.T) {
	for _, v := range []string{"", "all", "failed", "none"} {
		if err := ValidateKeepWorkdirs(v); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", v, err)
		}
	}
	if err := ValidateKeepWorkdirs("bogus"); err == nil {
		t.Error("expected error for invalid value \"bogus\"")
	}
}
