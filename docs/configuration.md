# Configuration

Suites are defined in YAML files. This page covers the full configuration schema.

## Suite Structure

```yaml
version: 1
description: "Suite description"
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  samples: 3
  timeout: 300
evals:
  - id: my-eval
    # ...
```

### Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `version` | Yes | Schema version (currently `1`) |
| `description` | No | Human-readable suite description |
| `defaults` | No | Default values inherited by all evals |
| `evals` | Yes | List of evaluations |

## Defaults

Defaults are inherited by all evals and can be overridden at the eval or treatment level.

```yaml
defaults:
  model: "claude-sonnet-4-6"
  runner: "claude-code"
  runner_config:
    allowed_tools:
      - "Read"
      - "Write"
  samples: 3
  timeout: 300
  parallel: 4
```

| Field | Description |
|-------|-------------|
| `model` | Model identifier |
| `runner` | Runner to use (`claude-code`, `ollama`) |
| `runner_config` | Runner-specific configuration (deep-merged) |
| `samples` | Number of runs per treatment |
| `timeout` | Timeout in seconds |
| `parallel` | Max concurrent samples per treatment (default: sequential) |
| `retry` | Retry configuration for failed runs (see [Retry](#retry)) |

## Evals

Each eval defines a task to evaluate.

```yaml
evals:
  - id: fizzbuzz
    prompt: "Write a FizzBuzz program in Go"
    dir: ./workspace
    isolate: true
    complexity: medium
    timeout: 120
    samples: 5
    model: "claude-sonnet-4-6"
    correctness:
      execute: true
    setup:
      before: "mkdir -p workspace"
      reset: "rm -rf workspace/*"
      after: "rm -rf workspace"
    treatments:
      control:
        name: baseline
      variations:
        - name: with-skill
          skill: "./skills/go-expert.md"
```

### Eval Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier for this eval |
| `name` | No | Human-readable display name for this eval |
| `prompt` | Yes | The task prompt sent to the AI agent |
| `dir` | No | Working directory for execution |
| `isolate` | No | Create a temporary copy of `dir` for each sample |
| `complexity` | No | Metadata: `low`, `medium`, or `high` |
| `timeout` | No | Override default timeout (seconds) |
| `samples` | No | Override default sample count |
| `parallel` | No | Override default max concurrency |
| `model` | No | Override default model |
| `runner` | No | Override default runner |
| `runner_config` | No | Runner-specific config (deep-merged with defaults) |
| `correctness` | No | Verification configuration (see [Verifiers](/verifiers)) |
| `setup` | No | Lifecycle hooks |
| `treatments` | Yes* | Treatment definitions (*or use `matrix`) |
| `matrix` | Yes* | Matrix dimensions (*or use `treatments`) |

### Prompt from File

For long prompts, reference an external file:

```yaml
evals:
  - id: complex-task
    prompt:
      file: ./prompts/complex-task.md
```

### Setup Hooks

```yaml
setup:
  before: "npm install"     # Run once before all samples
  reset: "git checkout ."   # Run before each sample
  after: "rm -rf node_modules"  # Run once after all samples
```

## Treatments

Treatments define the variants being compared. Every eval needs a `control` and zero or more `variations`.

```yaml
treatments:
  control:
    name: baseline
  variations:
    - name: with-skill
      skill: "./skills/my-skill.md"
    - name: with-skillset
      skills:
        - "./skills/skill-a.md"
        - "./skills/skill-b.md"
```

### Treatment Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique name for this treatment |
| `prompt` | No | Override the eval prompt |
| `skill` | No | Path to a single skill file |
| `skills` | No | List of skill file paths (concatenated) |
| `dir` | No | Override the eval working directory |
| `config_dir` | No | Sets `CLAUDE_CONFIG_DIR` environment variable |
| `model` | No | Override the model |
| `runner` | No | Override the runner |
| `runner_config` | No | Runner-specific config (deep-merged with defaults) |
| `env` | No | Environment variables for this treatment |
| `retry` | No | Override retry configuration |

### Override Precedence

Treatment > Eval > Defaults

For `runner_config`, values are deep-merged (treatment keys override, but unset keys are inherited).

## Matrix

Use `matrix` instead of `treatments` to generate a cartesian product of dimensions. This is useful for cross-cutting comparisons (e.g., model x skill).

```yaml
evals:
  - id: model-comparison
    prompt: "Solve the coding challenge"
    matrix:
      dimensions:
        - name: model
          values:
            - label: sonnet
              model: "claude-sonnet-4-6"
            - label: haiku
              model: "claude-haiku-4-5-20251001"
        - name: approach
          values:
            - label: baseline
            - label: with-skill
              skill: "./skills/expert.md"
```

This generates four treatments: `sonnet-baseline`, `sonnet-with-skill`, `haiku-baseline`, `haiku-with-skill`. The first combination becomes the control.

### Matrix Value Fields

Each value in a dimension can override any treatment-level field:

| Field | Description |
|-------|-------------|
| `label` | Required. Used to generate the treatment name |
| `prompt` | Override prompt |
| `model` | Override model |
| `runner` | Override runner |
| `skill` / `skills` | Skill injection |
| `env` | Environment variables |
| `runner_config` | Runner-specific config |

::: warning
`matrix` and `treatments` are mutually exclusive within the same eval.
:::

## Retry

Configure retry behavior for failed sample runs. By default, each sample runs once with no retries.

```yaml
defaults:
  retry:
    max_attempts: 3
    backoff: exponential
    delay: 2s
    on: transient
```

| Field | Default | Description |
|-------|---------|-------------|
| `max_attempts` | `1` | Total attempts including the first. `1` means no retries |
| `backoff` | `fixed` | `fixed` or `exponential`. Exponential doubles the delay each attempt |
| `delay` | `2s` | Base delay between retries (Go duration: `500ms`, `2s`, `1m`) |
| `on` | `transient` | `transient` or `all`. Controls which failures trigger retries |

### Retry Modes

- **`transient`** — Retry on runner errors, timeouts, and network failures. Don't retry if the agent ran successfully but produced incorrect output.
- **`all`** — Retry on any non-pass outcome, including correctness failures. Useful for flaky evals or giving the agent another chance.

### Backoff

- **`fixed`** — Waits the same `delay` between each attempt (with ±25% jitter).
- **`exponential`** — Doubles the delay each attempt: `delay`, `2×delay`, `4×delay`, etc. (with ±25% jitter).

### Result Selection

When multiple attempts are made, the **best** result is kept (not the last). Pass beats fail, fail beats error, and lower cost breaks ties.

### Override Precedence

Retry config inherits via the standard precedence: treatment > eval > defaults.

```yaml
defaults:
  retry:
    max_attempts: 2
evals:
  - id: flaky-eval
    retry:
      max_attempts: 5        # overrides defaults for this eval
      on: all
    treatments:
      control:
        name: baseline
      variations:
        - name: experimental
          retry:
            max_attempts: 3   # overrides eval for this treatment
```

## Runner Configuration

### claude-code

```yaml
runner_config:
  allowed_tools:
    - "Read"
    - "Write"
    - "Bash"
  disallowed_tools:
    - "WebSearch"
  mcp_config: "./mcp.json"
  max_budget_usd: 1.0
```

### ollama

```yaml
runner: ollama
runner_config:
  temperature: 0.7
  num_ctx: 4096
  num_predict: 2048
  top_p: 0.9
  top_k: 40
  seed: 42
  stop: ["\n\n"]
  think: true
```

## File References

Evals can be split into separate files:

```yaml
evals:
  - file: ./evals/fizzbuzz.yaml
  - file: ./evals/sorting.yaml
  - id: inline-eval
    prompt: "..."
```

The referenced YAML files contain the eval definition without the `evals` wrapper.
