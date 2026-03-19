package executor

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/driangle/skival/internal/suite"
)

// HookResult captures the outcome of running a lifecycle hook.
type HookResult struct {
	Stdout string
	Stderr string
	Err    error
}

// runHook executes a shell command in the given directory.
// Returns nil if the script is empty (no-op).
func runHook(ctx context.Context, script, dir string) *HookResult {
	if script == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	hr := &HookResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if ctx.Err() != nil {
			hr.Err = fmt.Errorf("hook timed out: %v", ctx.Err())
		} else {
			reason := strings.TrimSpace(stderr.String())
			if reason == "" {
				reason = err.Error()
			}
			hr.Err = fmt.Errorf("hook failed: %s", reason)
		}
	}

	return hr
}

// runSetupHooks manages the before/reset/after lifecycle for an eval.
// Returns a function to execute treatments, handling reset between them and after at the end.
func runBeforeHook(ctx context.Context, setup suite.Setup, dir string) error {
	if setup.Before != "" {
		slog.Debug("Running before hook", "script", setup.Before, "dir", dir)
	}
	if hr := runHook(ctx, setup.Before, dir); hr != nil && hr.Err != nil {
		return fmt.Errorf("setup.before: %w", hr.Err)
	}
	return nil
}

func runResetHook(ctx context.Context, setup suite.Setup, dir string) error {
	if setup.Reset != "" {
		slog.Debug("Running reset hook", "script", setup.Reset, "dir", dir)
	}
	if hr := runHook(ctx, setup.Reset, dir); hr != nil && hr.Err != nil {
		return fmt.Errorf("setup.reset: %w", hr.Err)
	}
	return nil
}

func runAfterHook(ctx context.Context, setup suite.Setup, dir string) {
	if setup.After != "" {
		slog.Debug("Running after hook", "script", setup.After, "dir", dir)
	}
	// After hook failures are warnings only — we don't propagate the error.
	hr := runHook(ctx, setup.After, dir)
	if hr != nil && hr.Err != nil {
		slog.Debug("After hook failed", "err", hr.Err)
	}
}
