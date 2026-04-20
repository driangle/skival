package verifier

import (
	"context"
	"encoding/json"
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
}

// Verifier checks whether a run's output meets correctness criteria.
type Verifier interface {
	Verify(ctx context.Context, input VerifyInput) VerifyResult
}
