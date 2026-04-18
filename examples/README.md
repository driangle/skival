# Example Suites

Self-contained example suites demonstrating the full range of skival configuration options. Each directory contains a `suite.yaml` plus any supporting files (skills, scripts, referenced evals).

| Directory | Description |
|-----------|-------------|
| `minimal/` | Simplest valid suite: one eval, two treatments (baseline vs. model comparison) |
| `defaults/` | Suite-level defaults (model, samples, timeout, runner, runner_config) inherited by evals |
| `file-refs/` | Evals loaded via `file:` references to separate YAML files |
| `multi-treatment/` | Control vs. multiple variations with different models, skills, env vars, and runner_config |
| `correctness/` | All correctness modes: compiles, execute, expected_output, script, state assertions, judge |
| `setup-hooks/` | Before/after/reset lifecycle hooks with isolation |
| `complexity/` | Evals at each complexity level (low, medium, high) with different sample counts |
| `runner-config/` | Runner and runner_config at defaults, eval, and treatment levels showing override precedence |
| `multi-runner/` | Different runners (claude-code, codex, aider) across treatments in the same suite |
| `fizzbuzz/` | Baseline vs. skill comparison with script-based verification |
| `matrix-comparison/` | Matrix syntax for cross-cutting runner x model evaluation |
| `per-treatment-config/` | Per-treatment prompt and config_dir overrides |
| `skillset-comparison/` | Single skill vs. composed skillset comparison |
