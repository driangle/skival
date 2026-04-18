package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CompilesVerifier checks that the code in a directory compiles successfully
// by running a user-provided build command.
type CompilesVerifier struct {
	Dir     string
	Command string
}

func (v *CompilesVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	c := exec.CommandContext(ctx, "sh", "-c", v.Command)
	c.Dir = v.Dir

	var stderr bytes.Buffer
	c.Stderr = &stderr

	err := c.Run()
	if err == nil {
		return VerifyResult{Pass: true, Reason: fmt.Sprintf("compiles: build succeeded (%s)", v.Command)}
	}

	if ctx.Err() != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("compiles: build timed out: %v", ctx.Err()),
		}
	}

	reason := strings.TrimSpace(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("compiles: build failed: %s", reason),
	}
}
