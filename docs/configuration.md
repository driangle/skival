# Configuration

Suites are defined in YAML files. This page covers the full configuration schema.

## Suite Structure

```yaml
version: 1
title: "Suite title"
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
| `title` | No | Short heading for the report (falls back to "Eval Report"); `description` renders as a subtitle beneath it |
| `description` | No | Human-readable suite description |
| `defaults` | No | Default values inherited by all evals |
| `ranking` | No | Ranking weight configuration (see [Ranking](#ranking)) |
| `compare` | No | Comparative judging configuration (see [Comparative Judging](#comparative-judging)) |
| `evals` | Yes | List of evaluations |

## Defaults

Defaults are inherited by all evals and can be overridden at the eval or variant level.

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
| `runner` | Runner to use (`claude-code`, `ollama`, `exec`) |
| `runner_config` | Runner-specific configuration (deep-merged) |
| `samples` | Number of runs per variant |
| `timeout` | Timeout in seconds |
| `parallel` | Max concurrent samples per variant (default: sequential) |
| `retry` | Retry configuration for failed runs (see [Retry](#retry)) |
| `judge_model` | Default model for the judge verifier (default: `claude-haiku-4-5-20251001`) |

### Runners

Built-in runners include `claude-code` and `ollama`. To evaluate an **arbitrary
program of your own** — in any language, calling any model or orchestration —
use the generic `exec` runner and describe the invocation in `runner_config`.
See [Exec Runner](/exec-runner) for the full contract (prompt delivery modes,
injected environment variables, and the optional JSONL event protocol).

## Evals

Each eval defines a task to evaluate.

```yaml
evals:
  - id: fizzbuzz
    prompt: "Write a FizzBuzz program in Go"
    dir: ./workspace
    isolate: true
    timeout: 120
    samples: 5
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    setup:
      before: "mkdir -p workspace"
      reset: "rm -rf workspace/*"
      after: "rm -rf workspace"
    variants:
      - name: baseline
      - name: with-skill
        skill: "./skills/go-expert.md"
```

### Eval Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier for this eval |
| `name` | No | Human-readable display name for this eval |
| `prompt` | Yes* | The task prompt sent to the AI agent (*or use `prompt_file`) |
| `prompt_file` | Yes* | Path to a file whose contents become the prompt (*or use `prompt`) |
| `vars` | No | Variables substituted into a `prompt_file` template (see below) |
| `dir` | No | Working directory for execution |
| `isolate` | No | Copy `dir` into a fresh temp dir per sample (default: `false`, see below) |
| `timeout` | No | Override default timeout (seconds) |
| `samples` | No | Override default sample count |
| `parallel` | No | Override default max concurrency |
| `model` | No | Override default model |
| `runner` | No | Override default runner |
| `runner_config` | No | Runner-specific config (deep-merged with defaults) |
| `verify` | No | Verification steps (see [Verifiers](/verifiers)) |
| `setup` | No | Lifecycle hooks |
| `variants` | Yes* | Variant definitions (*or use `matrix`) |
| `matrix` | Yes* | Matrix dimensions (*or use `variants`) |

### Working Directory Isolation

By default (`isolate: false`) every sample of an eval runs directly in the
configured `dir`. Samples share that directory, so a run that mutates files can
leak state into later samples of the same eval.

Set `isolate: true` to run each sample in its own throwaway copy of `dir`:

```yaml
evals:
  - id: fizzbuzz
    dir: ./workspace
    isolate: true      # each sample gets a fresh copy of ./workspace
    samples: 5
```

Notes:

- **Off by default.** Copying `dir` for every sample costs disk space and time
  proportional to the directory's size × sample count, so isolation is opt-in.
  Keep it off for read-only evals or ones with no `dir`.
- **No `dir`, no copy.** If neither the eval nor the variant sets `dir`, there is
  nothing to isolate and the flag is a no-op.
- **Copies are kept for inspection.** Each isolated copy is created under the
  system temp dir as `skival-isolate-*` and is *not* deleted after the run — its
  path appears in the report's **Workdirs** section so you can inspect what the
  agent produced. Clean these up yourself (e.g. `rm -rf $TMPDIR/skival-isolate-*`)
  when you no longer need them.

### Prompt from File

Long prompts bloat `suite.yaml`, lose syntax highlighting, and are awkward to diff. Use `prompt_file` to keep the prompt in a plain text/markdown file instead. Paths resolve relative to the suite file — or, for evals loaded via `file:`, relative to the eval file.

```yaml
evals:
  - id: complex-task
    prompt_file: prompts/complex-task.md
```

`prompt` and `prompt_file` are mutually exclusive at the same level; setting both is a load error. A `prompt_file` may be set on the eval (shared by all variants) or on an individual variant (which overrides the eval-level one).

#### Variable Substitution

A `prompt_file` may contain `{{name}}` placeholders filled from a `vars:` map. This parameterizes a single template across variants without duplication. Variant `vars` override eval `vars`:

```yaml
evals:
  - id: refactor
    prompt_file: prompts/refactor.md   # contains {{language}} / {{tone}}
    vars:
      language: "Go"
    variants:
      - name: strict
        vars:
          tone: "terse and precise"
      - name: verbose
        vars:
          tone: "detailed and explanatory"
```

Substitution only runs when at least one variable is defined; a template with no `vars` is used verbatim (literal `{{...}}` is preserved). When `vars` are defined, any placeholder left unresolved is a load error, so typos surface immediately instead of leaking into the prompt.

### Setup Hooks

```yaml
setup:
  before: "npm install"     # Run once before all samples
  reset: "git checkout ."   # Run before each sample
  after: "rm -rf node_modules"  # Run once after all samples
```

## Variants

Variants define the configurations being compared. Every eval needs at least one variant. The first variant in the list is treated as the baseline for ranking.

```yaml
variants:
  - name: baseline
  - name: with-skill
    skill: "./skills/my-skill.md"
  - name: with-skillset
    skills:
      - "./skills/skill-a.md"
      - "./skills/skill-b.md"
```

### Variant Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique name for this variant |
| `prompt` | No | Override the eval prompt |
| `prompt_file` | No | Override the eval prompt with a file's contents |
| `vars` | No | Variables merged over eval `vars` for template substitution |
| `skill` | No | Path to a single skill file |
| `skills` | No | List of skill file paths (concatenated) |
| `dir` | No | Override the eval working directory |
| `config_dir` | No | Sets `CLAUDE_CONFIG_DIR` environment variable |
| `model` | No | Override the model |
| `runner` | No | Override the runner |
| `runner_config` | No | Runner-specific config (deep-merged with defaults) |
| `env` | No | Environment variables for this variant |
| `retry` | No | Override retry configuration |

### Override Precedence

Variant > Eval > Defaults

For `runner_config`, values are deep-merged (variant keys override, but unset keys are inherited).

## Matrix

Use `matrix` instead of `variants` to generate a cartesian product of dimensions. This is useful for cross-cutting comparisons (e.g., model x skill).

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

This generates four variants: `sonnet-baseline`, `sonnet-with-skill`, `haiku-baseline`, `haiku-with-skill`. The first combination is treated as the baseline for ranking.

### Matrix Value Fields

Each value in a dimension can override any variant-level field:

| Field | Description |
|-------|-------------|
| `label` | Required. Used to generate the variant name |
| `prompt` | Override prompt |
| `model` | Override model |
| `runner` | Override runner |
| `skill` / `skills` | Skill injection |
| `env` | Environment variables |
| `runner_config` | Runner-specific config |

::: warning
`matrix` and `variants` are mutually exclusive within the same eval.
:::

## Ranking

Configure how variants are scored and ranked. The composite score is a weighted sum of correctness, cost, duration, tokens, and quality.

```yaml
ranking:
  weights:
    correctness: 0.60    # default: 0.60
    cost: 0.28           # default: 0.28
    duration: 0.12       # default: 0.12
    tokens: 0.00         # default: 0.00
    quality: 0.00        # default: 0.00
```

| Field | Default | Description |
|-------|---------|-------------|
| `weights.correctness` | `0.60` | Weight for pass rate (higher is better) |
| `weights.cost` | `0.28` | Weight for median cost (lower is better) |
| `weights.duration` | `0.12` | Weight for median duration (lower is better) |
| `weights.tokens` | `0.00` | Weight for median total tokens, input + output (lower is better) |
| `weights.quality` | `0.00` | Weight for comparative-judge quality (higher is better; see [Comparative Judging](#comparative-judging)) |

All weights must be `>= 0` and must sum to `1.0`. When the `ranking` section is omitted, the default weights apply. `tokens` and `quality` both default to `0.0`, so suites that set neither rank exactly as before. When you enable `compare` **without** setting explicit `ranking.weights`, a quality weight is carved out automatically (see [Comparative Judging](#comparative-judging)).

### Ranking on tokens instead of cost

Cost is only meaningful when skival knows the model's pricing. For local models, self-hosted or newly-released models, and the `exec` runner, the runner reports `cost_usd = 0`, so the cost dimension silently contributes nothing. Token usage is a **model-agnostic efficiency signal** — it lets you compare a token-hungry cheap model against a terse expensive one without trusting a price table.

To rank on tokens, move the economic weight from `cost` to `tokens`:

```yaml
ranking:
  weights:
    correctness: 0.60
    cost: 0.00           # no pricing available
    duration: 0.12
    tokens: 0.28         # rank efficiency by tokens instead
```

**Do not weight both `cost` and `tokens`.** Cost is approximately `tokens × price`, so the two measure the same underlying economic signal — giving both a nonzero weight double-counts it. Pick one: `cost` when you trust the pricing, `tokens` when you don't. skival does not forbid setting both (the weights just have to sum to `1.0`), so it is on you to choose one economic basis.

Variants whose runner reports no token usage are scored as `0` total tokens — i.e. treated as the most efficient, exactly the way a `cost = 0` variant is treated as free. A suite that ranks on tokens while mixing runners that report tokens with runners that don't will therefore favor the token-less ones; keep the compared variants on runners that all report usage.

### How the composite score is computed

- **Correctness** uses the pass rate directly (fraction of verified runs that passed), which is already on a `0–1` scale.
- **Cost, duration, and tokens are scored per eval, relative to that eval's best variant** (ratio-to-best): the cheapest/fastest/most-token-efficient variant in an eval scores `1.0`, one that is twice as expensive scores `0.5`, and so on. This makes the score sensitive to the *size* of a gap, not just who won — losing cost by 1% and by 90% produce different scores. The per-eval scores are then averaged across evals, so a cheap eval and an expensive eval are never pooled into a single figure.
- **Single-variant evals** score `1.0` on cost, duration, and tokens by definition (the lone variant is its own best), so only its pass rate can pull the composite below the weight sum.

The `median cost`, `median tokens`, and `median duration` shown in the rankings table are the mean of each variant's per-eval medians. The `median tokens` column appears only when `weights.tokens > 0`.

## Comparative Judging

Per-run verification answers "did this variant pass?" Comparative judging answers "which passing variant is *better*?" An LLM judge sees the outputs of the variants that passed an eval and scores each on a 1–5 quality scale against your criteria. The scores are normalized to `0–1` and fed into ranking through the `quality` weight.

Comparative judging is **opt-in** and **correctness-first**:

- It runs only as a **tiebreaker**, among the variants that passed every one of their per-run `verify` steps. A variant that failed can never win on quality, and an eval with fewer than two passing variants is skipped.
- It is **bounded**: one judge call per eval, regardless of how many variants are compared.
- It **degrades gracefully**: if the judge errors or returns unparseable output, that eval falls back to per-run pass/fail and the suite continues.

To reduce bias, the judge sees the outputs in a **shuffled order under anonymous labels** (Output A, Output B, …) — never the variant names.

```yaml
compare:
  criteria:
    - "explanation is clear and easy to follow"
    - "covers the key trade-offs"
  model: "claude-haiku-4-5-20251001"   # optional; defaults to defaults.judge_model or the built-in judge model
  max_chars: 4000                       # optional; truncate each output before judging (default 4000)
  weight: 0.2                           # optional; quality weight when ranking.weights is not set explicitly (default 0.15)
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `criteria` | Yes (when enabled) | — | The qualities the judge weighs when scoring outputs |
| `enabled` | No | `true` | Set to `false` to define a block without running it |
| `model` | No | `defaults.judge_model`, else built-in | Judge model for the comparison |
| `max_chars` | No | `4000` | Per-output truncation cap; set negative to disable truncation |
| `weight` | No | `0.15` | Quality weight used when `ranking.weights` is not set (suite level only) |

### Per-eval overrides

An eval can override or disable the suite-level block with its own `compare:` field. Eval-level fields (criteria, model, max_chars) take precedence; setting `enabled: false` disables comparison for that eval only.

```yaml
compare:
  criteria: ["clear and correct"]
evals:
  - id: sensitive-eval
    # ...
    compare:
      enabled: false          # skip comparison for this eval
  - id: prose-eval
    # ...
    compare:
      criteria: ["engaging and well-structured"]   # different criteria for this eval
```

### How quality feeds ranking

When comparison is enabled and you have **not** set explicit `ranking.weights`, skival allocates `compare.weight` (default `0.15`) to the `quality` dimension and renormalizes correctness/cost/duration to fill the rest, so the weights still sum to `1.0`. To control the balance precisely, set `ranking.weights` yourself (including a `quality` entry).

A variant's quality score is the mean of its per-eval comparative scores, averaged **only over the evals where it was compared** — a variant excluded from an eval's comparison (because it failed there) is not additionally penalized on quality, since its pass rate already reflects the failure.

Comparative scores appear per eval in all three report formats and, when persisted, as `comparison.json` under each eval's results directory.

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

Retry config inherits via the standard precedence: variant > eval > defaults.

```yaml
defaults:
  retry:
    max_attempts: 2
evals:
  - id: flaky-eval
    prompt: "..."
    runner: claude-code
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    retry:
      max_attempts: 5        # overrides defaults for this eval
      on: all
    variants:
      - name: baseline
      - name: experimental
        retry:
          max_attempts: 3   # overrides eval for this variant
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

`allowed_tools` is an **exclusive built-in whitelist**: any built-in tool not listed
(e.g. `Bash`, `Edit`) is denied for the run, including built-ins added in future CLI
releases — you never maintain a deny list. Scoped entries like `Bash(git:*)` are
matched by base name, and MCP tools (`mcp__*`) are governed by `mcp_config` rather
than this list. Enforcement uses the CLI's `--tools` flag and requires the `claude`
CLI **≥ 2.1.0**.

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
