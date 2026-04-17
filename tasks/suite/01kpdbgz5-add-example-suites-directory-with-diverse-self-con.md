---
title: "Add example suites directory with diverse self-contained examples"
id: "01kpdbgz5"
status: completed
priority: medium
type: feature
tags: ["docs", "suite"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Add example suites directory with diverse self-contained examples

## Objective

Create an `examples/` directory containing self-contained suite YAML files that demonstrate the full range of skival configuration options. These serve as both documentation and smoke-test fixtures — users can reference them to learn how to write suites, and CI can load them to catch regressions.

## Tasks

- [x] Create `examples/` directory at the repo root
- [x] `minimal.yaml` — simplest valid suite: one eval, one control treatment, inline prompt
- [x] `defaults.yaml` — suite-level defaults (model, samples, timeout, runner, runner_config) inherited by evals
- [x] `file-refs.yaml` — evals loaded via `file:` references to separate YAML files (include the referenced eval files)
- [x] `multi-treatment.yaml` — control vs. multiple variations with different models, skills, env vars, and runner_config
- [x] `correctness.yaml` — all correctness modes: compiles, execute, expected_output, script, state assertions, judge
- [x] `setup-hooks.yaml` — before/after/reset lifecycle hooks
- [x] `complexity.yaml` — evals at each complexity level (low, medium, high) with different sample counts
- [x] `runner-config.yaml` — runner and runner_config at defaults, eval, and treatment levels showing override precedence
- [x] `multi-runner.yaml` — different runners (claude-code, codex, aider) across treatments in the same suite
- [x] Add a README.md inside `examples/` briefly describing each file
- [x] Verify all examples load without errors via `go test` or a loader smoke test

## Acceptance Criteria

- Each example file is self-contained (or includes its referenced files in the same directory)
- Every suite struct field is exercised by at least one example
- All examples pass `Load()` without validation errors
- Examples are easy to scan and learn from (comments where helpful)
