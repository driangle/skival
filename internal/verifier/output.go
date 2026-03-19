package verifier

import (
	"context"
	"fmt"
	"strings"
)

// OutputVerifier checks that the run output contains all expected substrings.
type OutputVerifier struct {
	ExpectedSubstrings []string
}

func (v *OutputVerifier) Verify(_ context.Context, input VerifyInput) VerifyResult {
	for _, sub := range v.ExpectedSubstrings {
		if !strings.Contains(input.RunOutput, sub) {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("expected substring not found in output: %q", sub),
			}
		}
	}
	return VerifyResult{Pass: true, Reason: "all expected substrings found"}
}
