# skival — Implementation Plan

## Overview

skival is a Go CLI that benchmarks AI coding skills by running configurable eval suites. It measures four dimensions — time, tokens, cost, correctness — and compares N treatments against a control.

## Architecture

```
apps/cli/
  main.go                  # Entry point, cobra root command
  cmd/
    run.go                 # `skival run` command
    report.go              # `skival report` command

internal/
  suite/                   # Suite configuration parsing & validation
    suite.go               # Suite, Eval, Treatment types
    loader.go              # YAML loading and defaults
    validate.go            # Schema validation

  runner/                  # Agent execution abstraction
    runner.go              # Runner interface
    claude.go              # Claude Code runner (via agentrunner-go)

  result/                  # Result collection and aggregation
    result.go              # EvalResult, TreatmentResult, Metrics types
    aggregate.go           # Multi-sample aggregation (median, CV, min/max)

  verifier/                # Correctness checking
    verifier.go            # Verifier interface
    output.go              # Substring matching on stdout
    script.go              # Run a user-provided verification script
    state.go               # HTTP state assertions against a service
    judge.go               # LLM-as-judge for subjective criteria

  executor/                # Code execution
    executor.go            # Run generated code in a sandbox

  report/                  # Report generation
    markdown.go            # Markdown tables and summaries
    json.go                # JSON output
    rank.go                # Weighted scoring and ranking
```

## Core Concepts

### Suite

A suite is a YAML file defining a collection of evals and their treatments. Top-level structure:

```yaml
version: 1
description: "Suite description"

defaults:                        # Applied to all evals unless overridden
  samples: 3
  timeout: 60
  model: "claude-sonnet-4-20250514"

evals:
  - id: unique-eval-id
    name: "Human-readable name"
    prompt: "The task prompt sent to the agent"
    dir: "./evals/unique-eval-id"  # Working directory for this eval
    complexity: medium               # low | medium | high (metadata)
    timeout: 30                      # Per-run timeout in seconds

    setup:                           # Optional lifecycle hooks
      before: "npm install"          # Run before first treatment
      after: "docker-compose down"   # Run after all treatments
      reset: "npm run reset-db"      # Run between treatments

    correctness:
      compiles: "go build ./..."     # Build command (exit 0 = pass)
      agent_exits_ok: true           # Agent process exited with code 0
      expected_output:               # Substrings that must appear in stdout
        - "expected string"
      script: "./verify.sh"          # Custom verification script (exit 0 = pass)
      state:                         # HTTP assertions after execution
        - url: "http://localhost:3000/api/items"
          method: GET
          expect: "item_name"

    treatments:
      control:
        name: "baseline"
        dir: "./evals/unique-eval-id/baseline"
        env:                         # Extra env vars for this treatment
          SOME_VAR: "value"
      variations:
        - name: "with-skill"
          dir: "./evals/unique-eval-id/with-skill"
          skill: "./skills/my-skill"
          allowed_tools: ["Read", "Write", "Bash"]
        - name: "different-model"
          model: "claude-opus-4-20250514"
```

### Treatment

A treatment is a single configuration variant being evaluated. Every eval has exactly one **control** and zero or more **variations**. Each treatment can override:

- Working directory (different file context)
- Model
- Skill (Claude Code skill/CLAUDE.md to load)
- Allowed/disallowed tools
- System prompt or append-system-prompt
- Environment variables
- MCP server configuration

### Metrics

Each run produces:

| Metric | Source | Description |
|--------|--------|-------------|
| `wall_clock_ms` | Timer | Total elapsed time |
| `input_tokens` | agentrunner Result | Tokens consumed in prompt |
| `output_tokens` | agentrunner Result | Tokens generated |
| `cache_creation_tokens` | agentrunner Result | Tokens used to create cache |
| `cache_read_tokens` | agentrunner Result | Tokens read from cache |
| `cost_usd` | agentrunner Result | Dollar cost of the run |
| `num_turns` | Conversation history | Number of agentic turns |
| `pass` | Verifiers | Whether correctness criteria were met |

### Aggregation

When `samples > 1`, individual runs are aggregated:

- **pass**: All runs must pass (conservative — any failure means fail)
- **cost_usd, wall_clock_ms**: Median values reported
- **variance**: Min, max, coefficient of variation (CV) for cost and duration
- **individual_runs**: Each run's metrics stored for drill-down

CV is only computed for 3+ samples.

### Ranking

Treatments are ranked by weighted composite score:

| Factor | Weight | Description |
|--------|--------|-------------|
| Pass rate | 60% | Fraction of evals passed |
| Cost | 28% | Normalized inverse cost (lower is better) |
| Duration | 12% | Normalized inverse duration (lower is better) |

Scores are normalized across treatments so the best performer in each dimension gets 1.0.

## Runner Interface

```go
type Runner interface {
    Run(ctx context.Context, input RunInput) (RunOutput, error)
}

type RunInput struct {
    Prompt     string
    WorkingDir string
    Env        map[string]string
    Model      string
    Timeout    time.Duration
    // Claude-specific options passed through to agentrunner
    Options    map[string]any
}

type RunOutput struct {
    Text       string
    Metrics    Metrics
    History    []Message   // Full conversation for analysis
    SessionID  string
    ExitCode   int
    IsError    bool
}
```

The `ClaudeRunner` implementation wraps `agentrunner-go` (`github.com/driangle/agentrunner-go`). Future runners (Gemini CLI, Codex, etc.) implement the same interface.

## Verifier Interface

```go
type Verifier interface {
    Verify(ctx context.Context, input VerifyInput) (VerifyResult, error)
}

type VerifyInput struct {
    RunOutput    RunOutput
    Eval         Eval
    WorkingDir   string
}

type VerifyResult struct {
    Pass   bool
    Reason string
    Detail map[string]any
}
```

Built-in verifiers are composed into a pipeline: compile check -> agent_exits_ok -> output match -> state assertions -> script -> judge. Each step is optional based on the eval's `correctness` config. The pipeline short-circuits on first failure.

## Execution Flow

```
1. Load suite.yaml, validate
2. For each eval:
   a. Run setup.before (if defined)
   b. For each treatment (control first, then variations):
      i.   Run setup.reset (if not first treatment)
      ii.  For each sample 1..N:
           - Invoke runner with treatment config
           - Collect RunOutput (text, metrics, history)
           - Run verifier pipeline
           - Store TreatmentResult
      iii. Aggregate samples into single TreatmentResult
   c. Run setup.after (if defined)
   d. Store EvalResult with all treatment results
3. Rank treatments across all evals
4. Generate report (markdown/JSON) to stdout or results-dir
```

## Output

### Results Directory Structure

```
results/
  <timestamp>/
    summary.md                          # Aggregate ranking and comparison
    summary.json                        # Machine-readable results
    evals/
      <eval-id>/
        <treatment-name>/
          run-1.json                    # Per-run metrics and output
          run-1.conversation.jsonl      # Full conversation history
          run-2.json
          run-2.conversation.jsonl
        aggregate.json                  # Aggregated treatment results
```

### Markdown Report

The summary report includes:

1. **Results table** — One row per eval, columns per treatment, showing pass/fail + cost + duration
2. **Ranking table** — Treatments ranked by composite score with breakdown
3. **Per-treatment details** — Individual run metrics, aggregate stats, variance

## Implementation Phases

### Phase 1: Core Pipeline
- Suite YAML loading and validation
- Claude runner via agentrunner-go
- Basic execution flow (single sample, no verification)
- Console output of raw results

### Phase 2: Verification
- Output substring verifier
- Script verifier
- Execution of generated code
- State assertion verifier
- Verifier pipeline composition

### Phase 3: Aggregation & Reporting
- Multi-sample execution and aggregation
- Weighted ranking
- Markdown report generation
- JSON output
- Results directory persistence

### Phase 4: UX & Extensibility
- CLI flags for filtering evals/treatments
- Progress display during runs
- LLM judge verifier
- Setup lifecycle hooks (before/after/reset)
- Eval directory scaffolding command

### Future
- Additional runners (Gemini CLI, Codex, etc.)
- Parallel eval execution
- Comparison against historical baselines
- Dashboard / web UI for results
