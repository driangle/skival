package verifier

import (
	"context"
	"encoding/json"
	"os/exec"
)

// VerifyInput holds the data a verifier needs to check correctness.
type VerifyInput struct {
	// RunOutput is the text output produced by the run.
	RunOutput string
	// ExitCode is the process exit code from the run.
	ExitCode int
	// Conversation is the raw stream of session messages, when available.
	// The judge verifier uses it to include tool activity in its evaluation.
	Conversation []json.RawMessage
}

// VerifyResult holds the outcome of a verification check.
type VerifyResult struct {
	// Pass indicates whether the verification succeeded.
	Pass bool
	// Reason explains the result, especially useful on failure.
	Reason string
	// Conversation holds the raw JSON messages from the judge run, if any.
	Conversation []json.RawMessage
	// ExitCode is the process exit code, when the verifier ran a command. Nil
	// for verifier types that do not execute a subprocess.
	ExitCode *int
	// Stdout is the captured standard output of the verifier's command, if any.
	Stdout string
	// Stderr is the captured standard error of the verifier's command, if any.
	Stderr string
}

// Verifier checks whether a run's output meets correctness criteria.
type Verifier interface {
	Verify(ctx context.Context, input VerifyInput) VerifyResult
}

// exitCodeOf extracts the process exit code from an *exec.ExitError, returning
// nil when err is not an exit error (e.g. the command failed to launch).
func exitCodeOf(err error) *int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		return &code
	}
	return nil
}
