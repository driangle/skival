package exec

import (
	"context"
	"errors"
	"log/slog"
	osexec "os/exec"

	agentrunner "github.com/driangle/agentrunner/go"
)

// Runner is a stateless agentrunner.Runner that invokes an arbitrary command.
// Per-variant configuration flows through Options via WithConfig, so a single
// Runner instance can be cached and reused across every exec variant.
type Runner struct {
	logger *slog.Logger
}

// RunnerOption configures a Runner at construction time.
type RunnerOption func(*Runner)

// WithLogger sets the logger used for stderr and diagnostic output.
func WithLogger(l *slog.Logger) RunnerOption {
	return func(r *Runner) { r.logger = l }
}

// NewRunner constructs an exec Runner.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{logger: slog.Default()}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Start launches the configured command and returns a Session. Pre-flight
// errors (missing config, unresolvable binary) are returned immediately;
// runtime errors surface via Session.Result.
func (r *Runner) Start(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Session, error) {
	var options agentrunner.Options
	for _, o := range opts {
		o(&options)
	}

	cfg, err := configFromOptions(&options)
	if err != nil {
		return nil, err
	}

	var timeoutCancel context.CancelFunc
	if options.Timeout > 0 {
		ctx, timeoutCancel = context.WithTimeout(ctx, options.Timeout)
	}
	ctx, sessionCancel := context.WithCancel(ctx)

	inv, err := prepareInvocation(ctx, cfg, prompt, &options)
	if err != nil {
		sessionCancel()
		if timeoutCancel != nil {
			timeoutCancel()
		}
		return nil, err
	}

	stream := func(ctx context.Context, msgCh chan<- agentrunner.Message) (*agentrunner.Result, error) {
		if timeoutCancel != nil {
			defer timeoutCancel()
		}
		return r.run(ctx, inv, msgCh)
	}
	return agentrunner.NewSession(ctx, sessionCancel, stream), nil
}

// Run launches the command and blocks until it completes, draining any events.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	session, err := r.Start(ctx, prompt, opts...)
	if err != nil {
		return nil, err
	}
	for range session.Messages {
		// Drain so the streaming goroutine can complete.
	}
	return session.Result()
}

// run executes the prepared command, forwards session events, and assembles the
// result. A non-zero exit is a normal outcome reported via Result.ExitCode (so
// the agent_exits_ok verifier can judge it), not a runner error.
func (r *Runner) run(ctx context.Context, inv *invocation, msgCh chan<- agentrunner.Message) (*agentrunner.Result, error) {
	if inv.cleanup != nil {
		defer inv.cleanup()
	}

	if err := inv.cmd.Start(); err != nil {
		return nil, mapStartErr(err)
	}
	waitErr := inv.cmd.Wait()

	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, agentrunner.ErrTimeout
		}
		return nil, agentrunner.ErrCancelled
	}

	exitCode, err := exitCodeFrom(waitErr)
	if err != nil {
		return nil, err
	}

	final := forwardEvents(ctx, inv.eventsPath, msgCh)
	r.logStderr(inv)
	return buildResult(inv.stdout.String(), exitCode, final), nil
}

// mapStartErr translates a cmd.Start failure into a runner sentinel error.
func mapStartErr(err error) error {
	if errors.Is(err, osexec.ErrNotFound) {
		return errors.Join(agentrunner.ErrNotFound, err)
	}
	return err
}

// exitCodeFrom extracts the process exit code from a cmd.Wait error. A normal
// non-zero exit yields (code, nil); an unexpected wait failure yields (0, err).
func exitCodeFrom(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 0, err
}

// buildResult assembles a Result from captured stdout, the exit code, and an
// optional terminal final event. The final event's usage/cost fields populate
// the result; its text is a fallback when stdout is empty.
func buildResult(stdout string, exitCode int, final *finalEvent) *agentrunner.Result {
	res := &agentrunner.Result{
		Text:     stdout,
		ExitCode: exitCode,
		IsError:  exitCode != 0,
	}
	if final != nil {
		if res.Text == "" {
			res.Text = final.Text
		}
		res.CostUSD = final.CostUSD
		res.Usage = agentrunner.Usage{
			InputTokens:              final.Usage.InputTokens,
			OutputTokens:             final.Usage.OutputTokens,
			CacheCreationInputTokens: final.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     final.Usage.CacheReadInputTokens,
		}
	}
	return res
}

// logStderr logs the program's stderr output, if any, at debug level.
func (r *Runner) logStderr(inv *invocation) {
	if inv.stderr.Len() == 0 || r.logger == nil {
		return
	}
	r.logger.Debug("exec runner stderr", "output", inv.stderr.String())
}
