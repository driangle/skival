package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CheckOutputVerifier runs a shell command with the agent's text output piped to stdin.
// Exit code 0 means pass; non-zero means fail with stderr as the reason.
type CheckOutputVerifier struct {
	// Command is the shell command to execute.
	Command string
	// Dir is the working directory for the command.
	Dir string
}

func (v *CheckOutputVerifier) Verify(ctx context.Context, input VerifyInput) VerifyResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", v.Command)
	cmd.Dir = v.Dir
	cmd.Stdin = strings.NewReader(input.RunOutput)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return VerifyResult{Pass: true, Reason: "check_output exited 0"}
	}

	if ctx.Err() != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("check_output timed out: %v", ctx.Err()),
		}
	}

	reason := strings.TrimSpace(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("check_output failed: %s", reason),
	}
}
