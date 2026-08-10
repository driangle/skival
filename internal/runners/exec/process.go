package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	agentrunner "github.com/driangle/agentrunner/go"
)

// invocation bundles everything needed to run a single command and collect its
// output. It is built during Start (pre-flight) and executed by the streaming
// goroutine.
type invocation struct {
	cmd        *osexec.Cmd
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	eventsPath string
	// cleanup releases any temporary resources (e.g. the prompt file).
	cleanup func()
}

// prepareInvocation resolves the command, prompt delivery, environment, and
// output capture for a single run.
func prepareInvocation(ctx context.Context, cfg Config, prompt string, options *agentrunner.Options) (*invocation, error) {
	workDir := options.WorkingDir
	eventsPath := resolveEventsPath(cfg.EventsPath, workDir, options.Env)

	args, cleanup, err := resolveArgs(cfg, prompt)
	if err != nil {
		return nil, err
	}

	if err := preflightBinary(args[0]); err != nil {
		cleanup()
		return nil, err
	}

	cmd := osexec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Env = buildEnv(cfg, prompt, workDir, eventsPath, options.Env)
	if promptVia(cfg) == PromptViaStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	return &invocation{cmd: cmd, stdout: &stdout, stderr: &stderr, eventsPath: eventsPath, cleanup: cleanup}, nil
}

// resolveArgs returns the command args to execute and a cleanup func. In
// arg-file mode it writes the prompt to a temp file and substitutes the
// {prompt_file} placeholder; the cleanup removes that file.
func resolveArgs(cfg Config, prompt string) ([]string, func(), error) {
	noop := func() {}
	if promptVia(cfg) != PromptViaArgFile {
		return cfg.Command, noop, nil
	}
	path, args, err := writePromptFile(cfg.Command, prompt)
	if err != nil {
		return nil, noop, err
	}
	return args, func() { _ = os.Remove(path) }, nil
}

// preflightBinary reports a not-found error for bare command names that are not
// on PATH. Path-like commands are resolved relative to the working directory at
// start time, so they are not checked here.
func preflightBinary(name string) error {
	if strings.ContainsRune(name, '/') {
		return nil
	}
	if _, err := osexec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s", agentrunner.ErrNotFound, name)
	}
	return nil
}

// buildEnv composes the subprocess environment: the parent environment, the
// variant's env overrides, and the SKIVAL_* variables skival injects. Later
// entries win, so the injected variables take precedence.
func buildEnv(cfg Config, prompt, workDir, eventsPath string, extra map[string]string) []string {
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	env = append(env, DefaultPromptEnv+"="+prompt)
	if name := cfg.promptEnvName(); name != DefaultPromptEnv {
		env = append(env, name+"="+prompt)
	}
	if workDir != "" {
		env = append(env, "SKIVAL_RUN_DIR="+workDir)
	}
	if eventsPath != "" {
		env = append(env, "SKIVAL_EVENTS_PATH="+eventsPath)
	}
	return env
}

// resolveEventsPath expands ${SKIVAL_RUN_DIR} and other variables in the
// configured events path and anchors a relative result to the working dir.
func resolveEventsPath(raw, workDir string, extra map[string]string) string {
	if raw == "" {
		return ""
	}
	expanded := os.Expand(raw, func(key string) string {
		if key == "SKIVAL_RUN_DIR" {
			return workDir
		}
		if v, ok := extra[key]; ok {
			return v
		}
		return os.Getenv(key)
	})
	if !filepath.IsAbs(expanded) && workDir != "" {
		expanded = filepath.Join(workDir, expanded)
	}
	return expanded
}

// writePromptFile writes the prompt to a temp file and returns the file path
// plus a copy of command with the {prompt_file} placeholder replaced.
func writePromptFile(command []string, prompt string) (string, []string, error) {
	f, err := os.CreateTemp("", "skival-prompt-*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("creating prompt file: %w", err)
	}
	if _, err := f.WriteString(prompt); err != nil {
		f.Close()
		_ = os.Remove(f.Name())
		return "", nil, fmt.Errorf("writing prompt file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", nil, fmt.Errorf("closing prompt file: %w", err)
	}

	args := make([]string, len(command))
	for i, a := range command {
		if a == PromptFilePlaceholder {
			args[i] = f.Name()
		} else {
			args[i] = a
		}
	}
	return f.Name(), args, nil
}

// promptVia returns the effective prompt delivery mode, defaulting to stdin.
func promptVia(c Config) string {
	if c.PromptVia == "" {
		return PromptViaStdin
	}
	return c.PromptVia
}
