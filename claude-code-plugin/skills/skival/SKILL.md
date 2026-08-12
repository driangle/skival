---
name: skival
description: Generate skival suite.yaml files for evaluating and comparing AI agent configurations. Use when the user wants to create an eval suite, compare runners/models/skills/tool-access/environments, benchmark agent performance, or measure correctness/cost/speed across different AI setups.
---

You are an expert at creating skival eval suites. Skival evaluates and compares AI agent configurations by measuring correctness, cost, speed, and token usage across configurable variants. It answers questions like: "Does this skill file improve agent performance?", "Which model produces better results for this task?", "How does restricting tool access affect quality?", and "What's the cost/quality tradeoff between different configurations?"

## Your Task

Generate a valid `suite.yaml` file based on what the user wants to evaluate. Ask clarifying questions if the user's intent is ambiguous, but prefer sensible defaults over excessive questions.

**After writing the suite.yaml file, always validate it by running:**

```bash
skival validate <path-to-suite.yaml>
```

This parses the file, checks for structural errors, and prints a summary of evals, variants, and verifiers. If validation fails, fix the errors and re-validate until it passes.

## suite.yaml Schema

```yaml
version: 1                           # REQUIRED. Must be > 0. Always use 1.
title: "Short report heading"        # Optional. Report <h1>; falls back to "Eval Report".
description: "What this suite tests" # Optional. Renders as a subtitle beneath the title.

defaults:                             # Optional. Applied to all evals unless overridden.
  runner: claude-code                 # Default runner: claude-code | ollama | exec.
  samples: 3                          # Runs per variant (default 1; use 3+ for CV statistics).
  timeout: 60                         # Per-run timeout in seconds (default 30).
  parallel: 4                         # Default max concurrent samples.
  model: "claude-sonnet-4-6"          # Default model for all variants.
  judge_model: "claude-haiku-4-5-20251001"  # Default model for judge/compare steps.
  runner_config:                      # Default runner config, deep-merged into every variant.
    allowed_tools: ["Read", "Write", "Bash"]
  retry:                              # Default retry policy (see Retry below).
    max_attempts: 2

ranking:                              # Optional. Override composite scoring weights.
  weights:                            # All must be >= 0 and sum to exactly 1.0.
    correctness: 0.60
    cost: 0.28
    duration: 0.12
    quality: 0.00                     # Only meaningful when comparative judging runs.

compare:                              # Optional. Suite-level comparative judging (see below).
  criteria:
    - "Output is more readable"
  weight: 0.15                        # Ranking weight for quality (suite level only).

evals:                                # REQUIRED. At least one eval.
  - id: unique-eval-id               # REQUIRED. Unique kebab-case identifier.
    name: "Human-readable name"       # Optional display name.
    prompt: |                         # REQUIRED (unless every variant sets its own prompt).
      Write a function that...
    prompt_file: "./prompts/task.md"  # Optional. Load prompt from a file (mutually exclusive with prompt).
    vars:                             # Optional. {{var}} substitutions for the prompt template.
      LANG: "python"
    dir: "./evals/unique-eval-id"     # Optional. Working directory for this eval.
    isolate: true                     # Optional. Copy the working dir per sample (opt-in; has disk/time cost).
    samples: 5                        # Optional. Overrides defaults.samples.
    timeout: 120                      # Optional. Overrides defaults.timeout (seconds).
    parallel: 2                       # Optional. Overrides defaults.parallel.
    model: "claude-sonnet-4-6"        # Optional. Overrides defaults.model.
    runner: claude-code               # Optional. Overrides defaults.runner.
    runner_config: {}                 # Optional. Deep-merged over defaults, under each variant.
    retry: {}                         # Optional. Overrides defaults.retry.
    compare: {}                       # Optional. Per-eval comparative judging override.

    setup:                            # Optional. Lifecycle hooks (shell commands).
      before: "npm install"           # Runs ONCE before any variant starts.
      reset: "git checkout -- ."      # Runs BETWEEN variants (not before the first one).
      after: "docker-compose down"    # Runs ONCE after all variants complete.

    verify:                           # REQUIRED. At least one step; all must pass (see Verifiers).
      - type: agent_exits_ok

    variants:                         # REQUIRED (unless using `matrix`). At least one variant.
      - name: "baseline"              # REQUIRED. Unique name. The first variant is the ranking baseline.
        prompt: "..."                 # Optional. Variant-specific prompt (overrides the eval prompt).
        prompt_file: "./p.md"         # Optional. Variant-specific prompt file.
        vars: {}                      # Optional. Variant-specific template vars.
        dir: "./evals/eval-id/baseline"  # Optional. Override working directory.
        config_dir: "./config"        # Optional. Path to a config directory (must exist).
        model: "claude-sonnet-4-6"    # Optional. Override model.
        runner: claude-code           # Optional. Override runner.
        skill: "./skills/my-skill.md" # Optional. Path to a single skill file.
        skills:                       # Optional. Multiple skill files (mutually exclusive with skill).
          - "./skills/a.md"
          - "./skills/b.md"
        runner_config:                # Optional. Runner-specific config (deep-merged with defaults/eval).
          allowed_tools:              # e.g. restrict the toolset for this variant.
            - "Read"
            - "Write"
        env:                          # Optional. Environment variables.
          NODE_ENV: "test"
        retry: {}                     # Optional. Variant-specific retry policy.
      - name: "with-skill"            # Additional variants compared against the baseline.
        skill: "./skills/my-skill.md"
      - name: "different-model"
        model: "claude-opus-4-6"
```

Every value in `defaults` cascades into evals, and every value on an eval cascades into its variants; `runner_config` is deep-merged (variant keys win over eval keys win over defaults keys). A variant must ultimately resolve to a `runner` and (except for the `exec` runner) a `model`.

## Runners

Set `runner` at the defaults, eval, or variant level. Three runners are wired up:

- **`claude-code`** — the default. Model must start with `claude-`. `runner_config` accepts `allowed_tools`, `disallowed_tools`, `mcp_config`, and `max_budget_usd`.
- **`ollama`** — local models via Ollama.
- **`exec`** — drives an arbitrary program (any language/agent). Needs no model; the invocation contract lives in `runner_config`:

```yaml
- name: "my-agent"
  runner: exec
  runner_config:
    command: ["python", "agent.py"]   # REQUIRED. Program and its arguments.
    prompt_via: stdin                 # How the prompt is delivered: stdin (default) | env | arg-file.
    prompt_env: PROMPT                # Env var name when prompt_via: env.
    events_path: "${SKIVAL_RUN_DIR}/events.jsonl"  # Optional JSONL session-events file skival reads.
```

(`codex` and `aider` pass validation but are not registered in the default runner set — don't emit them.)

## Verifiers

`verify` is an ordered list of steps; **all must pass** for a sample to count as correct. At least one step is required per eval. The available types and their type-specific fields:

```yaml
verify:
  - type: agent_exits_ok            # Agent process exited with code 0. (no fields)

  - type: output_contains           # Substrings that MUST appear in the agent output.
    values: ["expected string"]

  - type: check                     # Custom script; exit 0 = pass. Runs in the working dir.
    run: "./verify.sh"              #   (must be a real command, not the literal true/false)

  - type: check_output              # Custom script with the agent's text output piped to stdin.
    run: "grep -q PASS"             #   Runs in the working dir; exit 0 = pass.

  - type: command                   # Run a command and assert on its exit code / stdout.
    run: "npm test"
    exits: 0                        #   Optional expected exit code.
    stdout_contains: "0 failing"    #   Optional required stdout substring.

  - type: file_contains             # Assert about a file on disk.
    path: "./out.txt"
    exists: true                    #   Optional. Whether the file must exist.
    contains: "done"               #   Optional required substring.

  - type: http_check                # HTTP assertion after execution.
    url: "http://localhost:3000/api/items"
    method: GET
    status: 200
    body_contains: "item_name"

  - type: tcp_check                 # TCP connectivity assertion.
    host: "localhost"
    port: 8080

  - type: judge                     # Subjective pass/fail criteria evaluated by an LLM.
    criteria:
      - "Code is well-documented"
      - "Solution is idiomatic"
    model: "claude-haiku-4-5-20251001"  # Optional judge model (defaults to judge_model).
```

Setting a field that doesn't belong to a step's `type` is a validation error — the schema is strict.

## Comparative Judging (`compare`)

Distinct from the `judge` verifier (which is per-variant pass/fail), a `compare` block scores the outputs of the variants that **passed** an eval against each other on a 1–5 quality scale, producing a ranking signal. It is opt-in and never affects suites that omit it. Configure it at the suite level (applies to all evals) and/or per eval:

```yaml
compare:
  enabled: true                 # A present block defaults to ON; set false to define criteria without running.
  criteria:                     # REQUIRED when enabled. What the judge weighs.
    - "Output is more readable and idiomatic"
  model: "claude-haiku-4-5-20251001"  # Optional judge model (defaults to judge_model).
  max_chars: 4000               # Optional per-output truncation cap; negative disables truncation.
  weight: 0.15                  # Optional, SUITE LEVEL ONLY. Ranking weight for quality (default 0.15).
```

When comparison runs, a **quality** weight is carved out of the composite score (default 0.15) and the correctness/cost/duration weights are renormalized to make room. Use the CLI `--compare` / `--no-compare` flags to force it on or off at run time.

## Retry (`retry`)

Configurable at the defaults, eval, or variant level to retry failed samples:

```yaml
retry:
  max_attempts: 3         # Total attempts including the first (>= 1, default 1).
  backoff: exponential    # "fixed" or "exponential" (default "fixed").
  delay: "2s"             # Base delay between retries, a Go duration (default "2s").
  on: transient           # "transient" (default) or "all".
```

## Validation Rules

These are enforced by skival and will cause errors if violated:

1. `version` must be > 0 (always use `1`)
2. At least one eval is required in `evals`
3. Each eval must have a non-empty `id`, unique across the suite
4. Each eval must have a prompt — set `prompt`/`prompt_file` on the eval, OR give every variant its own `prompt`/`prompt_file`
5. Each eval must have at least one `verify` step
6. Each eval must define at least one `variant` (or a `matrix`); `variants` and `matrix` are mutually exclusive
7. Each variant must have a non-empty `name`, unique within the eval
8. Every variant must resolve to a `runner` (and, except for `exec`, a `model`) via defaults/eval/variant
9. `skill` and `skills` are mutually exclusive on a variant
10. A variant's `config_dir`, if set, must point to an existing directory
11. `ranking.weights` must all be >= 0 and sum to exactly 1.0
12. An enabled `compare` block must define `criteria`; `compare.weight` must be between 0 and 1
13. Unknown or misspelled keys are rejected at load time — fix any key the validator flags

## Common Patterns

### Comparing models
Same task, different models — find the cost/quality sweet spot:
```yaml
variants:
  - name: "sonnet"
    model: "claude-sonnet-4-6"
  - name: "opus"
    model: "claude-opus-4-6"
  - name: "haiku"
    model: "claude-haiku-4-5-20251001"
```

### Comparing skills/instructions
Identical models, different guidance — measure how instructions affect output:
```yaml
variants:
  - name: "no-guidance"
  - name: "with-style-guide"
    skill: "./skills/style-guide.md"
  - name: "with-both-docs"
    skills:
      - "./skills/style-guide.md"
      - "./skills/arch-doc.md"
```

### Comparing tool access
Restrict which tools variants can use — test whether tool constraints improve or degrade performance:
```yaml
variants:
  - name: "all-tools"
  - name: "read-only"
    runner_config:
      allowed_tools: ["Read", "Glob", "Grep"]
  - name: "no-bash"
    runner_config:
      disallowed_tools: ["Bash"]
```

### Comparing environments
Same task in different project setups or with different environment variables:
```yaml
variants:
  - name: "default-env"
    dir: "./projects/baseline"
  - name: "strict-mode"
    dir: "./projects/baseline"
    env:
      STRICT_LINT: "true"
      CI: "true"
  - name: "alt-project"
    dir: "./projects/alternative"
```

### Combining dimensions
Use a `matrix` to compare multiple factors at once — skival generates the cartesian product of all dimensions as variants (mutually exclusive with `variants`). Each value's `label` names it; the remaining fields (`prompt`, `config_dir`, `model`, `runner`, `runner_config`, `skill`, `skills`, `env`) override the variant:
```yaml
matrix:
  dimensions:
    - name: model
      values:
        - label: sonnet
          model: "claude-sonnet-4-6"
        - label: opus
          model: "claude-opus-4-6"
    - name: skill
      values:
        - label: no-skill
        - label: with-skill
          skill: "./skills/best-practices.md"
```

### Multi-step verification
Combine verifiers for thorough correctness checking (steps run in order, all must pass):
```yaml
verify:
  - type: agent_exits_ok
  - type: command
    run: "npm test"
    exits: 0
  - type: file_contains
    path: "./dist/bundle.js"
    exists: true
  - type: judge
    criteria:
      - "Code handles edge cases"
      - "Error messages are helpful"
```

### Comparative quality ranking
Rank the passing variants by subjective quality on top of correctness/cost/speed:
```yaml
compare:
  criteria:
    - "Cleaner, more idiomatic code"
    - "Better error handling"
evals:
  - id: refactor
    prompt: "Refactor src/parser.js for readability."
    verify:
      - type: command
        run: "npm test"
        exits: 0
    variants:
      - name: "baseline"
      - name: "with-guide"
        skill: "./skills/style-guide.md"
```

## Guidelines

1. **Prompts should be self-contained.** The agent sees only the prompt, the working directory contents, and any skill file. Write prompts that fully describe the task.
2. **Use `setup.reset` for isolation.** If variants modify shared state (files, databases), use reset to restore a clean state between variants. For per-sample working-dir isolation, set `isolate: true` (opt-in; it copies the dir per sample).
3. **Use `setup.before`** to install dependencies or set up test fixtures.
4. **Start with 3+ samples** for statistical significance (the default is 1). Skival computes Coefficient of Variation (CV) only with 3+ samples.
5. **Set realistic timeouts.** Complex tasks may need 120-300s. Simple tasks can use 30-60s.
6. **Use `dir`** to point each eval or variant at a prepared working directory with starter code, test files, etc.
7. **Prefer `check`/`command` verification** for complex correctness checks. The script runs in the working directory and should exit 0 for pass; use `check_output` when you need the agent's text output on stdin.
8. **Use `judge` and `compare` sparingly.** They call an LLM for each criterion/output, adding cost. Best for subjective quality assessments.
9. **Keep eval IDs short and descriptive.** They become directory names in results output.
10. **Vary one dimension at a time when possible.** The clearest comparisons isolate a single variable (model, skill, tools, env). When combining dimensions, prefer a `matrix` so the combinations are named automatically.
11. **Use `env` for configuration that shouldn't be in code.** API keys, feature flags, debug modes — anything that changes behavior without changing the prompt or skill.

## CLI Commands

```bash
skival validate suite.yaml                      # Validate structure (always do this first).

skival run suite.yaml                            # Run the suite (markdown report to stdout).
skival run suite.yaml --samples 5                # Override sample count.
skival run suite.yaml --results-dir ./results    # Save detailed results for later report/compare.
skival run suite.yaml --evals eval-1,eval-2      # Run specific evals only.
skival run suite.yaml --variants baseline,v1     # Run specific variants only.
skival run suite.yaml --format json              # Output format: markdown (default), json, html.
skival run suite.yaml -p 4                        # Max concurrent samples (--parallel).
skival run suite.yaml --parallel-variants 2       # Max concurrent variants per eval (skips reset hook).
skival run suite.yaml --timeout 120              # Override all timeouts (seconds).
skival run suite.yaml --compare                   # Force comparative judging on where criteria exist.
skival run suite.yaml --no-compare                # Disable comparative judging even if configured.

skival report ./results/<run>                     # Regenerate a report from saved results (--format).
skival compare ./results/<baseline> ./results/<candidate>  # Diff two saved runs (--format).
```

## Ranking

Skival ranks variants by a weighted composite score. The defaults:
- **Correctness (60%)** — pass rate across evals
- **Cost (28%)** — lower is better (normalized)
- **Speed (12%)** — lower duration is better (normalized)

Override them with the `ranking.weights` block (must sum to 1.0). When comparative judging runs, a **quality** weight (default 0.15) is added and the other three are renormalized to make room. Results include median cost/duration, min/max ranges, and CV for 3+ samples.
