# Multi-Runner Support

## Problem

skival hardcodes `claudecode.NewRunner()` as the only agent runner. There is no way to compare different agent runners (Claude Code vs Codex vs Ollama) against the same eval suite. The `agentrunner` library already provides a generic `Runner` interface with multiple implementations, but skival has no mechanism to select between them.

## Goals

1. Allow suite authors to specify which runner a treatment uses via `runner` in suite.yaml
2. Support runner-specific configuration (e.g., `allowed_tools` for Claude Code, `temperature` for Ollama)
3. Enable cross-runner comparisons within a single suite run
4. Default to `claude-code` when no runner is specified (backward compatible)

## Non-Goals

- Adding new runner implementations to `agentrunner-go` (out of scope)
- Parallel execution of evals/treatments (separate concern)
- Auto-detection of available runners

## Design

### Suite Schema Changes

Add a `runner` field at the defaults, eval, and treatment levels. Add a `runner_config` map for runner-specific options.

```yaml
version: 1

defaults:
  runner: claude-code          # new: default runner for all treatments
  runner_config:               # new: default runner-specific options
    allowed_tools: ["Read", "Write", "Bash"]
  model: "claude-sonnet-4-20250514"

evals:
  - id: fizzbuzz
    prompt: "Write fizzbuzz.sh"
    treatments:
      control:
        name: "claude-code-baseline"
        # inherits runner: claude-code from defaults

      variations:
        - name: "claude-code-with-skill"
          skill: "./skills/shell.md"
          runner_config:
            allowed_tools: ["Read", "Write", "Bash"]

        - name: "ollama-local"
          runner: ollama
          model: "qwen3:8b"
          runner_config:
            temperature: 0.7
            num_ctx: 8192

        - name: "codex"
          runner: codex
          runner_config:
            sandbox: full
```

#### Precedence (highest to lowest)

| Field | Treatment | Eval | Defaults |
|-------|-----------|------|----------|
| `runner` | treatment.runner | eval.runner | defaults.runner |
| `runner_config` | deep-merged: treatment over eval over defaults | eval.runner_config | defaults.runner_config |
| `model` | treatment.model | eval.model | defaults.model |

`runner_config` values are **deep-merged** down the chain (treatment keys override eval keys which override default keys). This lets you set baseline runner config at the defaults level and override specific keys per treatment.

#### Built-in Runner Names

| Name | Package | Description |
|------|---------|-------------|
| `claude-code` | `agentrunner/go/claudecode` | Claude Code CLI agent (default) |
| `ollama` | `agentrunner/go/ollama` | Local models via Ollama API |

New runners can be added to the registry without schema changes.

### Type Changes

#### `suite.go`

```go
type Defaults struct {
    Samples      *int              `yaml:"samples"`
    Timeout      *int              `yaml:"timeout"`
    Model        string            `yaml:"model"`
    Runner       string            `yaml:"runner"`        // new
    RunnerConfig map[string]any    `yaml:"runner_config"` // new
}

type Eval struct {
    // ... existing fields ...
    Runner       string            `yaml:"runner"`        // new
    RunnerConfig map[string]any    `yaml:"runner_config"` // new
}

type Treatment struct {
    // ... existing fields ...
    Runner       string            `yaml:"runner"`        // new
    RunnerConfig map[string]any    `yaml:"runner_config"` // new
}
```

Remove `AllowedTools` from `Treatment` — it moves into `runner_config` since it is Claude Code-specific. Existing suites using `allowed_tools` at the treatment level continue to work via a backward-compat shim in the loader that migrates it into `runner_config.allowed_tools`.

#### `loader.go` — New Merge Logic

`mergeDefaults` gains runner/runner_config merging:

```go
func mergeDefaults(s *Suite) {
    d := s.Defaults
    for i := range s.Evals {
        e := &s.Evals[i]
        // ... existing merges ...

        if e.Runner == "" && d.Runner != "" {
            e.Runner = d.Runner
        }
        e.RunnerConfig = mergeMaps(d.RunnerConfig, e.RunnerConfig)
    }
}
```

A new `mergeEvalIntoTreatment` step resolves the final runner + runner_config for each treatment:

```go
func resolveRunnerConfig(eval *Eval, t *Treatment) {
    if t.Runner == "" {
        t.Runner = eval.Runner
    }
    t.RunnerConfig = mergeMaps(eval.RunnerConfig, t.RunnerConfig)
}
```

### Runner Registry

A new `internal/registry` package provides a factory for runner construction:

```go
package registry

import agentrunner "github.com/driangle/agentrunner/go"

// Factory creates a runner from a name and runner-specific config.
type Factory func(config map[string]any) (agentrunner.Runner, error)

// Registry maps runner names to their factories.
type Registry struct {
    factories map[string]Factory
}

func New() *Registry {
    return &Registry{factories: make(map[string]Factory)}
}

func (r *Registry) Register(name string, f Factory) {
    r.factories[name] = f
}

func (r *Registry) Create(name string, config map[string]any) (agentrunner.Runner, error) {
    f, ok := r.factories[name]
    if !ok {
        return nil, fmt.Errorf("unknown runner %q", name)
    }
    return f(config)
}
```

#### Default Registry Setup

In `apps/cli/cmd/run.go`:

```go
func defaultRegistry() *registry.Registry {
    r := registry.New()

    r.Register("claude-code", func(config map[string]any) (agentrunner.Runner, error) {
        return claudecode.NewRunner(claudecode.WithLogger(slog.Default())), nil
    })

    r.Register("ollama", func(config map[string]any) (agentrunner.Runner, error) {
        return ollama.NewRunner(), nil
    })

    return r
}
```

### Executor Changes

The executor currently takes a single `agentrunner.Runner`. It needs to accept the registry instead and resolve the runner per treatment.

```go
// Before
func Execute(ctx context.Context, s *suite.Suite, runner agentrunner.Runner, opts *Options) (*result.SuiteResult, error)

// After
func Execute(ctx context.Context, s *suite.Suite, reg *registry.Registry, opts *Options) (*result.SuiteResult, error)
```

Runner instantiation moves into `executeTreatment`. Runners are cached by name so the same runner type is reused across treatments:

```go
func executeTreatment(ctx context.Context, eval *suite.Eval, t *suite.Treatment, ..., reg *registry.Registry, cache map[string]agentrunner.Runner, ...) result.TreatmentResult {
    runnerName := t.Runner
    if runnerName == "" {
        runnerName = "claude-code"
    }

    runner, ok := cache[runnerName]
    if !ok {
        var err error
        runner, err = reg.Create(runnerName, t.RunnerConfig)
        if err != nil { /* handle */ }
        cache[runnerName] = runner
    }

    // ... rest of execution ...
}
```

#### `buildRunOptions` Changes

`buildRunOptions` must translate `runner_config` into the appropriate `agentrunner.Option` values based on the runner type. Runner-specific option mapping:

```go
func buildRunOptions(eval *suite.Eval, t *suite.Treatment, modelOverride string) ([]agentrunner.Option, error) {
    var opts []agentrunner.Option

    // ... model, dir, timeout, env (unchanged, these are runner-agnostic) ...

    // Skill file → appended system prompt (runner-agnostic)
    if t.Skill != "" { /* unchanged */ }

    // Runner-specific options from runner_config
    opts = append(opts, buildRunnerSpecificOpts(t.Runner, t.RunnerConfig)...)

    opts = append(opts, agentrunner.WithSkipPermissions())
    return opts, nil
}

func buildRunnerSpecificOpts(runner string, config map[string]any) []agentrunner.Option {
    switch runner {
    case "claude-code", "":
        return buildClaudeCodeOpts(config)
    case "ollama":
        return buildOllamaOpts(config)
    default:
        return nil
    }
}
```

**Claude Code option mapping** (`runner_config` key → agentrunner option):

| Config Key | Type | Maps To |
|-----------|------|---------|
| `allowed_tools` | `[]string` | `claudecode.WithAllowedTools(...)` |
| `disallowed_tools` | `[]string` | `claudecode.WithDisallowedTools(...)` |
| `mcp_config` | `string` | `claudecode.WithMCPConfig(path)` |
| `max_budget_usd` | `float64` | `claudecode.WithMaxBudgetUSD(n)` |

**Ollama option mapping** (`runner_config` key → agentrunner option):

| Config Key | Type | Maps To |
|-----------|------|---------|
| `temperature` | `float64` | `ollama.WithTemperature(n)` |
| `num_ctx` | `int` | `ollama.WithNumCtx(n)` |
| `num_predict` | `int` | `ollama.WithNumPredict(n)` |
| `top_p` | `float64` | `ollama.WithTopP(n)` |
| `top_k` | `int` | `ollama.WithTopK(n)` |
| `seed` | `int` | `ollama.WithSeed(n)` |
| `stop` | `[]string` | `ollama.WithStop(...)` |
| `think` | `bool` | `ollama.WithThink(b)` |

Unknown keys in `runner_config` produce a validation warning (not error) to allow forward compatibility.

### Validation Changes

Add to `validate.go`:

```go
var validRunners = map[string]bool{
    "":           true, // defaults to claude-code
    "claude-code": true,
    "ollama":      true,
}

// In validate():
if !validRunners[eval.Runner] {
    errs = append(errs, fmt.Sprintf("%s: unknown runner %q", prefix, eval.Runner))
}
// Same for each treatment.Runner
```

### Result Changes

Add runner name to results for reporting context:

```go
type TreatmentResult struct {
    Name      string       `json:"name"`
    Runner    string       `json:"runner"` // new
    IsControl bool         `json:"is_control"`
    Runs      []RunResult  `json:"runs"`
    Aggregate *Aggregate   `json:"aggregate,omitempty"`
}
```

### Reporting Changes

The markdown report should display the runner in the treatment header when multiple runners are used in a suite:

```
| Eval | baseline (claude-code) | ollama-local (ollama) | codex (codex) |
|------|------------------------|-----------------------|---------------|
| fizzbuzz | PASS $0.02 5.1s | PASS $0.00 12.3s | PASS $0.01 8.2s |
```

Rankings table gains a Runner column:

```
| Rank | Treatment | Runner | Score | Pass Rate | Cost | Duration |
|------|-----------|--------|-------|-----------|------|----------|
```

### Backward Compatibility

1. **No `runner` specified** → defaults to `claude-code`. Existing suites work unchanged.
2. **`allowed_tools` at treatment level** → loader migrates to `runner_config.allowed_tools` during loading (with deprecation log warning). This shim can be removed in version 2.
3. **Single runner `Execute` signature** → removed. Callers must pass a registry. The CLI is the only caller.

### CLI Changes

No new flags needed. The runner is selected via suite.yaml, not the CLI. The existing `--model` flag continues to override model for all treatments regardless of runner.

## File Changes

| File | Change |
|------|--------|
| `internal/suite/suite.go` | Add `Runner`, `RunnerConfig` to `Defaults`, `Eval`, `Treatment`. Remove `AllowedTools`. |
| `internal/suite/loader.go` | Merge runner/runner_config in defaults. Migrate `allowed_tools` shim. Add `resolveRunnerConfig`. |
| `internal/suite/validate.go` | Validate runner names. |
| `internal/registry/registry.go` | New file: runner registry with `Register`/`Create`. |
| `internal/executor/executor.go` | Accept `*registry.Registry` instead of `agentrunner.Runner`. Cache runners. |
| `internal/executor/executor.go` | `buildRunOptions` delegates to per-runner option builders. |
| `internal/result/result.go` | Add `Runner` field to `TreatmentResult`. |
| `internal/report/markdown.go` | Show runner name in headers when multiple runners present. |
| `internal/report/rank.go` | Include runner name in ranking output. |
| `apps/cli/cmd/run.go` | Build default registry, pass to executor. |

## Example: Cross-Runner Comparison Suite

```yaml
version: 1
description: "Compare Claude Code vs Ollama on shell scripting tasks"

defaults:
  samples: 3
  timeout: 120
  runner: claude-code
  model: "claude-sonnet-4-20250514"

evals:
  - id: fizzbuzz
    prompt: |
      Write fizzbuzz.sh that prints FizzBuzz for 1-20.
    complexity: low
    setup:
      reset: "rm -f fizzbuzz.sh"
    correctness:
      execute: true
      script: "./verify.sh"

    treatments:
      control:
        name: "claude-sonnet"

      variations:
        - name: "claude-opus"
          model: "claude-opus-4-20250514"

        - name: "ollama-qwen3"
          runner: ollama
          model: "qwen3:32b"
          runner_config:
            temperature: 0.3
            num_ctx: 16384

        - name: "claude-with-skill"
          skill: "./skills/shell-best-practices.md"
          runner_config:
            allowed_tools: ["Read", "Write", "Bash"]
```

## Testing

- **Unit**: Registry create/lookup, runner_config merging, per-runner option building, backward compat `allowed_tools` migration
- **Validation**: Unknown runner names rejected, runner_config type mismatches caught
- **Integration**: Suite loading with runner fields, full execution with mock runners
- **Backward compat**: Existing suite.yaml files without `runner` field load and execute as before
