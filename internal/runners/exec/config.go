// Package exec implements a generic agentrunner.Runner that invokes an
// arbitrary user-specified command with the eval prompt. It makes no
// assumptions about the user's language, framework, or model — the invocation
// is described entirely in the suite's runner_config.
package exec

import (
	"fmt"

	agentrunner "github.com/driangle/agentrunner/go"
)

// Prompt delivery modes. These describe how skival hands the eval prompt to
// the user's program.
const (
	// PromptViaStdin writes the prompt to the program's standard input.
	PromptViaStdin = "stdin"
	// PromptViaEnv passes the prompt through an environment variable
	// (SKIVAL_PROMPT by default, or the configured prompt_env).
	PromptViaEnv = "env"
	// PromptViaArgFile writes the prompt to a temp file and substitutes its
	// path for the {prompt_file} placeholder in the command arguments.
	PromptViaArgFile = "arg-file"
)

// PromptFilePlaceholder is the token replaced with the prompt file path when
// prompt_via is "arg-file".
const PromptFilePlaceholder = "{prompt_file}"

// DefaultPromptEnv is the environment variable used to deliver the prompt in
// "env" mode when prompt_env is not set. It is also always injected so
// programs can read the prompt regardless of the configured delivery mode.
const DefaultPromptEnv = "SKIVAL_PROMPT"

// Config describes how to invoke a user program for an exec-runner variant.
type Config struct {
	// Command is the program and its arguments, e.g. ["python", "agent.py"].
	Command []string
	// PromptVia selects the prompt delivery mode: stdin (default), env, or arg-file.
	PromptVia string
	// PromptEnv overrides the environment variable name used in "env" mode.
	PromptEnv string
	// EventsPath, when set, is the path skival advertises via SKIVAL_EVENTS_PATH
	// and reads JSONL session events from after the program exits. It may
	// reference ${SKIVAL_RUN_DIR} and other environment variables.
	EventsPath string
}

// configKey is the private key under which the resolved Config is stored in
// agentrunner Options' extra map.
type configKey struct{}

// WithConfig returns an Option that carries the exec Config for a single
// invocation. The runner itself is stateless; per-variant configuration flows
// through Options so a single cached runner instance serves every exec variant.
func WithConfig(c Config) agentrunner.Option {
	return func(o *agentrunner.Options) {
		o.SetExtra(configKey{}, c)
	}
}

// configFromOptions extracts the exec Config from the resolved Options.
func configFromOptions(o *agentrunner.Options) (Config, error) {
	v, ok := o.GetExtra(configKey{})
	if !ok {
		return Config{}, fmt.Errorf("exec runner: missing config (set runner_config.command)")
	}
	c, ok := v.(Config)
	if !ok {
		return Config{}, fmt.Errorf("exec runner: invalid config type %T", v)
	}
	if len(c.Command) == 0 {
		return Config{}, fmt.Errorf("exec runner: command is required")
	}
	return c, nil
}

// ParseConfig converts a runner_config map into a validated Config. It is the
// single source of truth for the exec runner_config shape and is used both to
// build runtime options and to validate suites.
func ParseConfig(m map[string]any) (Config, error) {
	var c Config

	cmd, err := stringSlice(m["command"])
	if err != nil {
		return Config{}, fmt.Errorf("command: %w", err)
	}
	c.Command = cmd

	if v, ok := m["prompt_via"]; ok {
		s, ok := v.(string)
		if !ok {
			return Config{}, fmt.Errorf("prompt_via must be a string")
		}
		c.PromptVia = s
	}
	if v, ok := m["prompt_env"]; ok {
		s, ok := v.(string)
		if !ok {
			return Config{}, fmt.Errorf("prompt_env must be a string")
		}
		c.PromptEnv = s
	}
	if v, ok := m["events_path"]; ok {
		s, ok := v.(string)
		if !ok {
			return Config{}, fmt.Errorf("events_path must be a string")
		}
		c.EventsPath = s
	}
	return c, nil
}

// Validate checks a parsed Config for structural problems, returning a list of
// human-readable error strings (empty when valid).
func (c Config) Validate() []string {
	var errs []string
	if len(c.Command) == 0 {
		errs = append(errs, "command is required and must be a non-empty list of strings")
	}
	switch c.PromptVia {
	case "", PromptViaStdin, PromptViaEnv:
		// ok
	case PromptViaArgFile:
		if !hasPlaceholder(c.Command) {
			errs = append(errs, fmt.Sprintf("prompt_via %q requires a %q placeholder in command", PromptViaArgFile, PromptFilePlaceholder))
		}
	default:
		errs = append(errs, fmt.Sprintf("prompt_via must be one of %q, %q, %q; got %q",
			PromptViaStdin, PromptViaEnv, PromptViaArgFile, c.PromptVia))
	}
	return errs
}

// promptEnvName returns the environment variable used for prompt delivery.
func (c Config) promptEnvName() string {
	if c.PromptEnv != "" {
		return c.PromptEnv
	}
	return DefaultPromptEnv
}

func hasPlaceholder(args []string) bool {
	for _, a := range args {
		if a == PromptFilePlaceholder {
			return true
		}
	}
	return false
}

// stringSlice coerces a YAML-decoded value into []string. It accepts []string
// and []any (the shape yaml.v3 produces), rejecting anything else.
func stringSlice(v any) ([]string, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	case []string:
		return val, nil
	case []any:
		out := make([]string, 0, len(val))
		for i, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("element %d is not a string", i)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("must be a list of strings")
	}
}
