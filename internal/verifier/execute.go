package verifier

import (
	"context"
	"fmt"
)

// ExecuteVerifier checks that the run completed with exit code 0.
type ExecuteVerifier struct{}

func (v *ExecuteVerifier) Verify(_ context.Context, input VerifyInput) VerifyResult {
	if input.ExitCode != 0 {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("non-zero exit code: %d", input.ExitCode),
		}
	}
	return VerifyResult{Pass: true, Reason: "exit code 0"}
}
