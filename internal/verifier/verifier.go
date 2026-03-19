package verifier

import "context"

// VerifyInput holds the data a verifier needs to check correctness.
type VerifyInput struct {
	// RunOutput is the text output produced by the run.
	RunOutput string
	// ExitCode is the process exit code from the run.
	ExitCode int
}

// VerifyResult holds the outcome of a verification check.
type VerifyResult struct {
	// Pass indicates whether the verification succeeded.
	Pass bool
	// Reason explains the result, especially useful on failure.
	Reason string
}

// Verifier checks whether a run's output meets correctness criteria.
type Verifier interface {
	Verify(ctx context.Context, input VerifyInput) VerifyResult
}
