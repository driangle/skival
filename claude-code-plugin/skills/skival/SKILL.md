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
description: "What this suite tests" # Optional.

defaults:                             # Optional. Applied to all evals unless overridden.
  runner: claude-code                 # Default runner for all evals.
  samples: 3                          # Runs per variant (more = better statistics, min 3 for CV).
  timeout: 60                         # Per-run timeout in seconds.
  model: "claude-sonnet-4-6"          # Default model for all variants.

evals:                                # REQUIRED. At least one eval.
  - id: unique-eval-id               # REQUIRED. Unique kebab-case identifier.
    name: "Human-readable name"       # Optional display name.
    prompt: |                         # REQUIRED. The task prompt sent to the agent.
      Write a function that...
    dir: "./evals/unique-eval-id"     # Optional. Working directory for this eval.
    samples: 5                        # Optional. Overrides defaults.samples.
    timeout: 120                      # Optional. Overrides defaults.timeout (seconds).
    model: "claude-sonnet-4-6"        # Optional. Overrides defaults.model.

    setup:                            # Optional. Lifecycle hooks (shell commands).
      before: "npm install"           # Runs ONCE before any variant starts.
      reset: "git checkout -- ."      # Runs BETWEEN variants (not before the first one).
      after: "docker-compose down"    # Runs ONCE after all variants complete.

    verify:                           # Optional. Ordered verification steps (all must pass).
      - type: agent_exits_ok          # Agent process exited with code 0.
      - type: output_contains         # Substrings that MUST appear in the agent output.
        values:
          - "expected string"
      - type: check_output            # Custom script; exit 0 = pass (agent output piped to stdin).
        run: "./verify.sh"
      - type: http_check              # HTTP assertion after execution.
        url: "http://localhost:3000/api/items"
        method: GET
        status: 200
        body_contains: "item_name"
      - type: judge                   # Subjective criteria evaluated by an LLM.
        criteria:
          - "Code is well-documented"
          - "Solution is idiomatic"

    variants:                         # REQUIRED (unless using `matrix`). At least one variant.
      - name: "baseline"              # REQUIRED. Unique name. The first variant is the ranking baseline.
        dir: "./evals/eval-id/baseline"  # Optional. Override working directory.
        model: "claude-sonnet-4-6"    # Optional. Override model.
        skill: "./skills/my-skill.md" # Optional. Path to a single skill file.
        runner_config:                # Optional. Runner-specific config (deep-merged with defaults).
          allowed_tools:              # e.g. restrict the toolset for this variant.
            - "Read"
            - "Write"
            - "Bash"
        env:                          # Optional. Environment variables.
          NODE_ENV: "test"
      - name: "with-skill"            # Additional variants compared against the baseline.
        skill: "./skills/my-skill.md"
      - name: "different-model"
        model: "claude-opus-4-6"
```

## Validation Rules

These are enforced by skival and will cause errors if violated:

1. `version` must be > 0 (always use `1`)
2. At least one eval is required in `evals`
3. Each eval must have a non-empty `id` (unique across the suite)
4. Each eval must have a non-empty `prompt`
5. Each eval must define at least one `variant` (or a `matrix`), and `variants` and `matrix` are mutually exclusive
6. Each variant must have a non-empty `name`, unique within the eval
7. Unknown or misspelled keys are rejected at load time — fix any key the validator flags

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
  - name: "with-architecture-doc"
    skill: "./skills/arch-doc.md"
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
      allowed_tools: ["Read", "Write", "Edit", "Glob", "Grep"]
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
Use a `matrix` to compare multiple factors at once — skival generates the cartesian product of all dimensions as variants:
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
  - type: output_contains
    values:
      - "All tests passed"
  - type: check_output
    run: "./verify.sh"
  - type: judge
    criteria:
      - "Code handles edge cases"
      - "Error messages are helpful"
```

## Guidelines

1. **Prompts should be self-contained.** The agent sees only the prompt, the working directory contents, and any skill file. Write prompts that fully describe the task.
2. **Use `setup.reset` for isolation.** If variants modify shared state (files, databases), use reset to restore a clean state between variants.
3. **Use `setup.before`** to install dependencies or set up test fixtures.
4. **Start with 3+ samples** for statistical significance. Skival computes Coefficient of Variation (CV) only with 3+ samples.
5. **Set realistic timeouts.** Complex tasks may need 120-300s. Simple tasks can use 30-60s.
6. **Use `dir`** to point each eval or variant at a prepared working directory with starter code, test files, etc.
7. **Prefer `check_output` verification** for complex correctness checks. The script runs in the working directory and should exit 0 for pass.
8. **Use `judge` sparingly.** It calls an LLM for each criterion, adding cost. Best for subjective quality assessments.
9. **Keep eval IDs short and descriptive.** They become directory names in results output.
10. **Vary one dimension at a time when possible.** The clearest comparisons isolate a single variable (model, skill, tools, env). When combining dimensions, prefer a `matrix` so the combinations are named automatically.
11. **Use `env` for configuration that shouldn't be in code.** API keys, feature flags, debug modes — anything that changes behavior without changing the prompt or skill.

## Running the Suite

After generating the suite.yaml, validate then run:

```bash
skival validate suite.yaml                      # Validate structure (always do this first)
skival run suite.yaml                           # Basic run
skival run suite.yaml --samples 5               # Override sample count
skival run suite.yaml --results-dir ./results   # Save detailed results
skival run suite.yaml --evals eval-1,eval-2     # Run specific evals only
skival run suite.yaml --variants baseline,v1    # Run specific variants only
skival run suite.yaml --format json             # JSON output instead of markdown
```

## Ranking

Skival ranks variants by a weighted composite score:
- **Correctness (60%)** - pass rate across evals
- **Cost (28%)** - lower is better (normalized)
- **Speed (12%)** - lower duration is better (normalized)

Results include median cost/duration, min/max ranges, and CV for 3+ samples.
