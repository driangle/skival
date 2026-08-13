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

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	if err == nil {
		return VerifyResult{
			Pass:   true,
			Reason: fmt.Sprintf("check: command succeeded (%s)", v.Command),
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}
	}

	if ctx.Err() != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("check: command timed out: %v", ctx.Err()),
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}
	}

	reason := strings.TrimSpace(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	return VerifyResult{
		Pass:     false,
		Reason:   fmt.Sprintf("check: command failed: %s", reason),
		ExitCode: exitCodeOf(err),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
}
