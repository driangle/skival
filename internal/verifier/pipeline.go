package verifier

import (
	"context"

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

// BuildPipeline assembles a verification pipeline from the eval's correctness config.
// Only verifiers whose config fields are present are included.
// Returns nil if no verifiers are configured.
// The runner and prompt are only needed when judge criteria are configured.
func BuildPipeline(c suite.Correctness, evalDir string, opts ...PipelineOption) *Pipeline {
	var cfg pipelineConfig
	for _, o := range opts {
		o(&cfg)
	}
	var steps []namedVerifier

	if c.Compiles != "" {
		steps = append(steps, namedVerifier{
			name:     "compiles",
			verifier: &CompilesVerifier{Dir: evalDir, Command: c.Compiles},
		})
	}

	if c.Execute != nil && *c.Execute {
		steps = append(steps, namedVerifier{
			name:     "execute",
			verifier: &ExecuteVerifier{},
		})
	}

	if len(c.ExpectedOutput) > 0 {
		steps = append(steps, namedVerifier{
			name:     "output",
			verifier: &OutputVerifier{ExpectedSubstrings: c.ExpectedOutput},
		})
	}

	if len(c.State) > 0 {
		assertions := make([]StateAssertion, len(c.State))
		for i, s := range c.State {
			assertions[i] = StateAssertion{
				URL:    s.URL,
				Method: s.Method,
				Expect: s.Expect,
			}
		}
		steps = append(steps, namedVerifier{
			name:     "state",
			verifier: &StateVerifier{Assertions: assertions},
		})
	}

	if c.Script != "" {
		steps = append(steps, namedVerifier{
			name:     "script",
			verifier: &ScriptVerifier{Script: c.Script, Dir: evalDir},
		})
	}

	if len(c.Judge) > 0 && cfg.runner != nil {
		steps = append(steps, namedVerifier{
			name: "judge",
			verifier: &JudgeVerifier{
				Runner:   cfg.runner,
				Criteria: c.Judge,
				Prompt:   cfg.evalPrompt,
				Model:    cfg.judgeModel,
			},
		})
	}

	if len(steps) == 0 {
		return nil
	}

	return &Pipeline{steps: steps}
}

// PipelineOption configures optional pipeline behavior.
type PipelineOption func(*pipelineConfig)

type pipelineConfig struct {
	runner     agentrunner.Runner
	evalPrompt string
	judgeModel string
}

// WithJudge provides a runner, eval prompt, and optional model for the judge verifier.
func WithJudge(runner agentrunner.Runner, evalPrompt string, judgeModel string) PipelineOption {
	return func(c *pipelineConfig) {
		c.runner = runner
		c.evalPrompt = evalPrompt
		c.judgeModel = judgeModel
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
