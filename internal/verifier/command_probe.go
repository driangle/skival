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

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("command timed out: %v", ctx.Err()),
			}
		} else {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("command failed to run: %v", err),
			}
		}
	}

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
