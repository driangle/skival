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
		name := step.Name

		switch step.Type {
		case "agent_exits_ok":
			if name == "" {
				name = "agent_exits_ok"
			}
			steps = append(steps, namedVerifier{
				name:     name,
				verifier: &ExecuteVerifier{},
			})
		case "check":
			if name == "" {
				name = "check"
			}
			steps = append(steps, namedVerifier{
				name:     name,
				verifier: &CheckVerifier{Dir: evalDir, Command: step.Run},
			})
		case "check_output":
			if name == "" {
				name = "check_output"
			}
			steps = append(steps, namedVerifier{
				name:     name,
				verifier: &CheckOutputVerifier{Command: step.Run, Dir: evalDir},
			})
		case "output_contains":
			if name == "" {
				name = "output_contains"
			}
			steps = append(steps, namedVerifier{
				name:     name,
				verifier: &OutputVerifier{ExpectedSubstrings: step.Values},
			})
		case "command":
			if name == "" {
				name = fmt.Sprintf("command[%d]", i)
			}
			steps = append(steps, namedVerifier{
				name: name,
				verifier: &CommandProbeVerifier{
					Probe: suite.CommandProbe{
						Run: step.Run,
						Assert: suite.CommandProbeAssert{
							Exits:          step.Exits,
							StdoutContains: step.StdoutContains,
						},
					},
					Dir: evalDir,
				},
			})
		case "file_contains":
			if name == "" {
				name = fmt.Sprintf("file_contains[%d]", i)
			}
			steps = append(steps, namedVerifier{
				name: name,
				verifier: &FileProbeVerifier{
					Probe: suite.FileProbe{
						Path: step.Path,
						Assert: suite.FileProbeAssert{
							Exists:   step.Exists,
							Contains: step.Contains,
						},
					},
				},
			})
		case "http_check":
			if name == "" {
				name = fmt.Sprintf("http_check[%d]", i)
			}
			steps = append(steps, namedVerifier{
				name: name,
				verifier: &HTTPProbeVerifier{
					Probe: suite.HTTPProbe{
						URL:    step.URL,
						Method: step.Method,
						Assert: suite.HTTPProbeAssert{
							Status:       step.Status,
							BodyContains: step.BodyContains,
						},
					},
				},
			})
		case "tcp_check":
			if name == "" {
				name = fmt.Sprintf("tcp_check[%d]", i)
			}
			steps = append(steps, namedVerifier{
				name: name,
				verifier: &TCPProbeVerifier{
					Probe: suite.TCPProbe{
						Host: step.Host,
						Port: step.Port,
					},
				},
			})
		case "judge":
			if cfg.runner != nil {
				if name == "" {
					name = "judge"
				}
				steps = append(steps, namedVerifier{
					name: name,
					verifier: &JudgeVerifier{
						Runner:   cfg.runner,
						Criteria: step.Criteria,
						Prompt:   cfg.evalPrompt,
						Model:    step.Model,
					},
				})
			}
		}
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
}

// WithJudge provides a runner and eval prompt for the judge verifier.
func WithJudge(runner agentrunner.Runner, evalPrompt string) PipelineOption {
	return func(c *pipelineConfig) {
		c.runner = runner
		c.evalPrompt = evalPrompt
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
