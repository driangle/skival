---
name: skival
description: Generate skival suite.yaml files for benchmarking AI agent skills. Use when the user wants to create an eval suite, benchmark skills, compare prompt strategies, compare models, or measure agent correctness/cost/speed on specific tasks.
---

You are an expert at creating skival eval suites. Skival benchmarks AI agent skills by measuring correctness, cost, speed, and token usage across configurable treatments. It answers questions like: "Does this skill file improve agent performance?", "Which prompt strategy produces better results?", and "How does model choice affect quality and cost for this task?"

## Your Task

Generate a valid `suite.yaml` file based on what the user wants to evaluate. Ask clarifying questions if the user's intent is ambiguous, but prefer sensible defaults over excessive questions.

## suite.yaml Schema

```yaml
version: 1                           # REQUIRED. Must be > 0. Always use 1.
description: "What this suite tests" # Optional.

defaults:                             # Optional. Applied to all evals unless overridden.
  samples: 3                          # Runs per treatment (more = better statistics, min 3 for CV).
  timeout: 60                         # Per-run timeout in seconds.
  model: "claude-sonnet-4-20250514"   # Default model for all treatments.

evals:                                # REQUIRED. At least one eval.
  - id: unique-eval-id               # REQUIRED. Unique kebab-case identifier.
    name: "Human-readable name"       # Optional display name.
    prompt: |                         # REQUIRED. The task prompt sent to the agent.
      Write a function that...
    dir: "./evals/unique-eval-id"     # Optional. Working directory for this eval.
    complexity: medium                # Optional. One of: low, medium, high.
    samples: 5                        # Optional. Overrides defaults.samples.
    timeout: 120                      # Optional. Overrides defaults.timeout (seconds).
    model: "claude-sonnet-4-20250514" # Optional. Overrides defaults.model.

    setup:                            # Optional. Lifecycle hooks (shell commands).
      before: "npm install"           # Runs ONCE before any treatment starts.
      reset: "git checkout -- ."      # Runs BETWEEN treatments (not before the first one).
      after: "docker-compose down"    # Runs ONCE after all treatments complete.

    correctness:                      # Optional. How to verify the agent's output.
      compiles: true                  # Check if generated code compiles.
      execute: true                   # Execute the generated code.
      expected_output:                # Substrings that MUST appear in stdout.
        - "expected string"
      script: "./verify.sh"           # Custom script (exit 0 = pass, non-zero = fail).
      state:                          # HTTP assertions after execution.
        - url: "http://localhost:3000/api/items"
          method: GET
          expect: "item_name"
      judge:                          # Subjective criteria evaluated by Claude Haiku.
        - "Code is well-documented"
        - "Solution is idiomatic"

    treatments:
      control:                        # REQUIRED. The baseline treatment.
        name: "baseline"              # REQUIRED. Unique treatment name.
        dir: "./evals/eval-id/baseline"  # Optional. Override working directory.
        model: "claude-sonnet-4-20250514" # Optional. Override model.
        skill: "./skills/my-skill"    # Optional. Path to a CLAUDE.md / skill file.
        allowed_tools:                # Optional. Whitelist of tools for this treatment.
          - "Read"
          - "Write"
          - "Bash"
        env:                          # Optional. Environment variables.
          NODE_ENV: "test"

      variations:                     # Optional. Zero or more treatment variants.
        - name: "with-skill"
          skill: "./skills/my-skill"
        - name: "different-model"
          model: "claude-opus-4-20250514"
```

## Validation Rules

These are enforced by skival and will cause errors if violated:

1. `version` must be > 0 (always use `1`)
2. At least one eval is required in `evals`
3. Each eval must have a non-empty `id` (unique across the suite)
4. Each eval must have a non-empty `prompt`
5. `complexity` must be one of: `low`, `medium`, `high` (or omitted)
6. `control.name` is required and must be non-empty
7. All treatment names within an eval should be unique

## Common Patterns

### Comparing models
Use the same prompt/correctness with different models in treatments:
```yaml
treatments:
  control:
    name: "sonnet"
    model: "claude-sonnet-4-20250514"
  variations:
    - name: "opus"
      model: "claude-opus-4-20250514"
    - name: "haiku"
      model: "claude-haiku-4-5-20251001"
```

### Comparing prompts/skills
Use identical models but different skill files:
```yaml
treatments:
  control:
    name: "no-guidance"
  variations:
    - name: "with-style-guide"
      skill: "./skills/style-guide/CLAUDE.md"
    - name: "with-architecture-doc"
      skill: "./skills/arch-doc/CLAUDE.md"
```

### Comparing tool access
Restrict which tools treatments can use:
```yaml
treatments:
  control:
    name: "all-tools"
  variations:
    - name: "read-only"
      allowed_tools: ["Read", "Glob", "Grep"]
    - name: "no-bash"
      allowed_tools: ["Read", "Write", "Edit", "Glob", "Grep"]
```

### Multi-step verification
Combine verifiers for thorough correctness checking:
```yaml
correctness:
  execute: true
  expected_output:
    - "All tests passed"
  script: "./verify.sh"
  judge:
    - "Code handles edge cases"
    - "Error messages are helpful"
```

## Guidelines

1. **Prompts should be self-contained.** The agent sees only the prompt, the working directory contents, and any skill file. Write prompts that fully describe the task.
2. **Use `setup.reset` for isolation.** If treatments modify shared state (files, databases), use reset to restore a clean state between treatments.
3. **Use `setup.before`** to install dependencies or set up test fixtures.
4. **Start with 3+ samples** for statistical significance. Skival computes Coefficient of Variation (CV) only with 3+ samples.
5. **Set realistic timeouts.** Complex tasks may need 120-300s. Simple tasks can use 30-60s.
6. **Use `dir`** to point each eval or treatment at a prepared working directory with starter code, test files, etc.
7. **Prefer `script` verification** for complex correctness checks. The script receives the working directory and should exit 0 for pass.
8. **Use `judge` sparingly.** It calls Claude Haiku for each criterion, adding cost. Best for subjective quality assessments.
9. **Keep eval IDs short and descriptive.** They become directory names in results output.

## Running the Suite

After generating the suite.yaml, the user runs it with:

```bash
skival run suite.yaml                           # Basic run
skival run suite.yaml --samples 5               # Override sample count
skival run suite.yaml --results-dir ./results   # Save detailed results
skival run suite.yaml --evals eval-1,eval-2     # Run specific evals only
skival run suite.yaml --treatments control,v1   # Run specific treatments only
skival run suite.yaml --model claude-opus-4     # Override model for all treatments
skival run suite.yaml --format json             # JSON output instead of markdown
```

## Ranking

Skival ranks treatments by a weighted composite score:
- **Correctness (60%)** - pass rate across evals
- **Cost (28%)** - lower is better (normalized)
- **Speed (12%)** - lower duration is better (normalized)

Results include median cost/duration, min/max ranges, and CV for 3+ samples.
