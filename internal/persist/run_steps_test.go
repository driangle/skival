package persist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/result"
)

func intPtr(i int) *int { return &i }

// makeSuiteWithFailingStep builds a suite whose run recorded a failing verify
// step carrying exit code, stdout, and stderr detail.
func makeSuiteWithFailingStep() *result.SuiteResult {
	sr := makeSuiteResult()
	sr.Evals[0].Variants[0].Runs[0].Pass = boolPtr(false)
	sr.Evals[0].Variants[0].Runs[0].Steps = []result.StepResult{
		{Name: "build", Type: "check", Pass: true, Reason: "check: command succeeded"},
		{
			Name:     "tests",
			Type:     "check",
			Pass:     false,
			ExitCode: intPtr(2),
			Stdout:   "running tests\n",
			Stderr:   "FAIL: expected 3 got 4\n",
			Reason:   "check: command failed: FAIL: expected 3 got 4",
		},
	}
	return sr
}

func TestSaveAndLoad_StepsRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteWithFailingStep()

	outDir, err := Save(tmpDir, sr, defaultWeights(), SaveOptions{})
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// The persisted run-N.json is exactly what a failure investigation reads.
	data, err := os.ReadFile(filepath.Join(outDir, "evals", "eval1", "control", "run-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"steps"`, `"exit_code": 2`, `"stderr"`, `FAIL: expected 3 got 4`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("run-1.json missing %q:\n%s", want, data)
		}
	}

	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	steps := loaded.Evals[0].Variants[0].Runs[0].Steps
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	fail := steps[1]
	if fail.Name != "tests" || fail.Type != "check" {
		t.Errorf("step name/type = %q/%q, want tests/check", fail.Name, fail.Type)
	}
	if fail.Pass {
		t.Error("failing step should have Pass=false")
	}
	if fail.ExitCode == nil || *fail.ExitCode != 2 {
		t.Errorf("exit code = %v, want 2", fail.ExitCode)
	}
	if fail.Stdout != "running tests\n" {
		t.Errorf("stdout = %q", fail.Stdout)
	}
	if fail.Stderr != "FAIL: expected 3 got 4\n" {
		t.Errorf("stderr = %q", fail.Stderr)
	}
	if fail.Reason != "check: command failed: FAIL: expected 3 got 4" {
		t.Errorf("reason = %q", fail.Reason)
	}
}

func TestSave_OmitsStepsWhenAbsent(t *testing.T) {
	tmpDir := t.TempDir()
	sr := makeSuiteResult() // runs carry no steps

	outDir, err := Save(tmpDir, sr, defaultWeights(), SaveOptions{})
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "evals", "eval1", "control", "run-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"steps"`) {
		t.Errorf("run-1.json should omit steps when absent:\n%s", data)
	}

	loaded, err := Load(outDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if steps := loaded.Evals[0].Variants[0].Runs[0].Steps; steps != nil {
		t.Errorf("expected nil steps for legacy run, got %+v", steps)
	}
}
