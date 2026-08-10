package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/driangle/skival/internal/suite"
)

// CommandProbeVerifier runs a shell command and checks exit code and stdout.
type CommandProbeVerifier struct {
	Probe suite.CommandProbe
	Dir   string
}

func (v *CommandProbeVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", v.Probe.Run)
	if v.Dir != "" {
		cmd.Dir = v.Dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode, failure := runCommandProbe(ctx, cmd)
	if failure != nil {
		return *failure
	}

	return v.checkAssertions(exitCode, &stdout, &stderr)
}

// runCommandProbe runs cmd and returns its exit code. It returns a non-nil
// failure result only when the command could not be run to completion (timeout
// or launch failure), leaving assertion checks to the caller.
func runCommandProbe(ctx context.Context, cmd *exec.Cmd) (int, *VerifyResult) {
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	if ctx.Err() != nil {
		return 0, &VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("command timed out: %v", ctx.Err()),
		}
	}
	return 0, &VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("command failed to run: %v", err),
	}
}

func (v *CommandProbeVerifier) checkAssertions(exitCode int, stdout, stderr *bytes.Buffer) VerifyResult {
	a := v.Probe.Assert

	if a.Exits != nil && exitCode != *a.Exits {
		reason := fmt.Sprintf("expected exit code %d, got %d", *a.Exits, exitCode)
		if stderr.Len() > 0 {
			reason += ": " + strings.TrimSpace(stderr.String())
		}
		return VerifyResult{Pass: false, Reason: reason}
	}

	if a.StdoutContains != "" && !strings.Contains(stdout.String(), a.StdoutContains) {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("expected %q not found in command stdout", a.StdoutContains),
		}
	}

	return VerifyResult{Pass: true, Reason: fmt.Sprintf("command probe passed: %s", v.Probe.Run)}
}
