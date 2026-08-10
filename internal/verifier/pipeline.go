package verifier

import (
	"context"
	"fmt"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/suite"
)

// StepResult records the outcome of a single pipeline step.
type StepResult struct {
	Name   string
	Result VerifyResult
}

// PipelineResult holds the outcome of running the full verification pipeline.
type PipelineResult struct {
	Pass  bool
	Steps []StepResult
}

// Pipeline runs verifiers in sequence, short-circuiting on first failure.
type Pipeline struct {
	steps []namedVerifier
}

type namedVerifier struct {
	name     string
	verifier Verifier
}

// BuildPipeline assembles a verification pipeline from verify steps.
// Steps run in list order. Returns nil if no steps are provided.
// The runner and prompt are only needed when judge steps are present.
func BuildPipeline(verifySteps []suite.VerifyStep, evalDir string, opts ...PipelineOption) *Pipeline {
	var cfg pipelineConfig
	for _, o := range opts {
		o(&cfg)
	}

	var steps []namedVerifier
	for i, step := range verifySteps {
		if nv, ok := buildStepVerifier(step, i, evalDir, cfg); ok {
			steps = append(steps, nv)
		}
	}

	if len(steps) == 0 {
		return nil
	}

	return &Pipeline{steps: steps}
}

// named builds a namedVerifier, defaulting the name to fallback when unset.
func named(name, fallback string, v Verifier) namedVerifier {
	if name == "" {
		name = fallback
	}
	return namedVerifier{name: name, verifier: v}
}

// buildStepVerifier constructs the verifier for a single verify step. The bool
// is false when the step type produces no verifier (unknown type, or a judge
// step with no runner configured).
func buildStepVerifier(step suite.VerifyStep, i int, evalDir string, cfg pipelineConfig) (namedVerifier, bool) {
	switch step.Type {
	case "agent_exits_ok":
		return named(step.Name, "agent_exits_ok", &ExecuteVerifier{}), true
	case "check":
		return named(step.Name, "check", &CheckVerifier{Dir: evalDir, Command: step.Run}), true
	case "check_output":
		return named(step.Name, "check_output", &CheckOutputVerifier{Command: step.Run, Dir: evalDir}), true
	case "output_contains":
		return named(step.Name, "output_contains", &OutputVerifier{ExpectedSubstrings: step.Values}), true
	case "judge":
		return buildJudgeVerifier(step, cfg)
	default:
		return buildProbeVerifier(step, i, evalDir)
	}
}

// buildProbeVerifier constructs probe-style verifiers whose default names are
// suffixed with the step index. It returns false for unrecognized types.
func buildProbeVerifier(step suite.VerifyStep, i int, evalDir string) (namedVerifier, bool) {
	switch step.Type {
	case "command":
		return named(step.Name, fmt.Sprintf("command[%d]", i), &CommandProbeVerifier{
			Probe: suite.CommandProbe{
				Run:    step.Run,
				Assert: suite.CommandProbeAssert{Exits: step.Exits, StdoutContains: step.StdoutContains},
			},
			Dir: evalDir,
		}), true
	case "file_contains":
		return named(step.Name, fmt.Sprintf("file_contains[%d]", i), &FileProbeVerifier{
			Probe: suite.FileProbe{
				Path:   step.Path,
				Assert: suite.FileProbeAssert{Exists: step.Exists, Contains: step.Contains},
			},
			Dir: evalDir,
		}), true
	case "http_check":
		return named(step.Name, fmt.Sprintf("http_check[%d]", i), &HTTPProbeVerifier{
			Probe: suite.HTTPProbe{
				URL:    step.URL,
				Method: step.Method,
				Assert: suite.HTTPProbeAssert{Status: step.Status, BodyContains: step.BodyContains},
			},
		}), true
	case "tcp_check":
		return named(step.Name, fmt.Sprintf("tcp_check[%d]", i), &TCPProbeVerifier{
			Probe: suite.TCPProbe{Host: step.Host, Port: step.Port},
		}), true
	default:
		return namedVerifier{}, false
	}
}

// buildJudgeVerifier constructs a judge verifier, returning false when no runner
// is configured (judge steps are silently skipped in that case).
func buildJudgeVerifier(step suite.VerifyStep, cfg pipelineConfig) (namedVerifier, bool) {
	if cfg.runner == nil {
		return namedVerifier{}, false
	}
	return named(step.Name, "judge", &JudgeVerifier{
		Runner:     cfg.runner,
		Criteria:   step.Criteria,
		Prompt:     cfg.evalPrompt,
		Model:      step.Model,
		AgentModel: cfg.agentModel,
	}), true
}

// PipelineOption configures optional pipeline behavior.
type PipelineOption func(*pipelineConfig)

type pipelineConfig struct {
	runner     agentrunner.Runner
	evalPrompt string
	agentModel string
}

// WithJudge provides a runner and eval prompt for the judge verifier.
func WithJudge(runner agentrunner.Runner, evalPrompt string) PipelineOption {
	return func(c *pipelineConfig) {
		c.runner = runner
		c.evalPrompt = evalPrompt
	}
}

// WithAgentModel tells the judge which model the agent-under-test ran as, so
// it can evaluate criteria that reference the agent's model without guessing
// from its own identity.
func WithAgentModel(model string) PipelineOption {
	return func(c *pipelineConfig) {
		c.agentModel = model
	}
}

// Run executes all verifiers in order, stopping on the first failure.
func (p *Pipeline) Run(ctx context.Context, input VerifyInput) PipelineResult {
	var completed []StepResult

	for _, s := range p.steps {
		r := s.verifier.Verify(ctx, input)
		completed = append(completed, StepResult{Name: s.name, Result: r})
		if !r.Pass {
			return PipelineResult{Pass: false, Steps: completed}
		}
	}

	return PipelineResult{Pass: true, Steps: completed}
}
