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

// HookError is an error that includes the captured stdout/stderr from a failed hook.
type HookError struct {
	Message string
	Stdout  string
	Stderr  string
}

func (e *HookError) Error() string {
	var parts []string
	parts = append(parts, e.Message)
	if e.Stderr != "" {
		parts = append(parts, "stderr: "+strings.TrimSpace(e.Stderr))
	}
	if e.Stdout != "" {
		parts = append(parts, "stdout: "+strings.TrimSpace(e.Stdout))
	}
	return strings.Join(parts, "\n")
}

// runHook executes a shell command in the given directory.
// Returns nil if the script is empty (no-op).
func runHook(ctx context.Context, script, dir string) *HookError {
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
	if err == nil {
		return nil
	}

	he := &HookError{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if ctx.Err() != nil {
		he.Message = fmt.Sprintf("hook timed out: %v", ctx.Err())
	} else {
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = err.Error()
		}
		he.Message = fmt.Sprintf("hook failed: %s", reason)
	}

	return he
}

func runBeforeHook(ctx context.Context, setup suite.Setup, dir string) error {
	if setup.Before != "" {
		slog.Debug("Running before hook", "script", setup.Before, "dir", dir)
	}
	if he := runHook(ctx, setup.Before, dir); he != nil {
		return fmt.Errorf("setup.before: %w", he)
	}
	return nil
}

func runResetHook(ctx context.Context, setup suite.Setup, dir string) error {
	if setup.Reset != "" {
		slog.Debug("Running reset hook", "script", setup.Reset, "dir", dir)
	}
	if he := runHook(ctx, setup.Reset, dir); he != nil {
		return fmt.Errorf("setup.reset: %w", he)
	}
	return nil
}

func runAfterHook(ctx context.Context, setup suite.Setup, dir string) {
	if setup.After != "" {
		slog.Debug("Running after hook", "script", setup.After, "dir", dir)
	}
	// After hook failures are warnings only — we don't propagate the error.
	if he := runHook(ctx, setup.After, dir); he != nil {
		slog.Debug("After hook failed", "err", he)
	}
}
