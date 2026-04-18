package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CheckVerifier checks that a shell command succeeds in the eval directory.
type CheckVerifier struct {
	Dir     string
	Command string
}

func (v *CheckVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	c := exec.CommandContext(ctx, "sh", "-c", v.Command)
	c.Dir = v.Dir

	var stderr bytes.Buffer
	c.Stderr = &stderr

	err := c.Run()
	if err == nil {
		return VerifyResult{Pass: true, Reason: fmt.Sprintf("check: command succeeded (%s)", v.Command)}
	}

	if ctx.Err() != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("check: command timed out: %v", ctx.Err()),
		}
	}

	reason := strings.TrimSpace(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("check: command failed: %s", reason),
	}
}
