package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ScriptVerifier runs a shell script to verify correctness.
// Exit code 0 means pass; non-zero means fail with stderr as the reason.
type ScriptVerifier struct {
	// Script is the shell command to execute.
	Script string
	// Dir is the working directory for the script.
	Dir string
}

func (v *ScriptVerifier) Verify(ctx context.Context, input VerifyInput) VerifyResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", v.Script)
	cmd.Dir = v.Dir
	cmd.Stdin = strings.NewReader(input.RunOutput)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return VerifyResult{Pass: true, Reason: "script exited 0"}
	}

	if ctx.Err() != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("script timed out: %v", ctx.Err()),
		}
	}

	reason := strings.TrimSpace(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("script failed: %s", reason),
	}
}
