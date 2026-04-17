---
title: "Per-treatment prompt and config_dir support"
id: "01kpeg6kw"
status: completed
priority: high
type: feature
tags: ["enhancement", "treatments", "configuration"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Per-treatment prompt and config_dir support

## Objective

Enable treatments to fully customize how Claude Code runs by supporting per-treatment `prompt` and `config_dir` fields. Currently, all treatments in an eval share the same prompt, and there is no way to point Claude Code at a different configuration directory per treatment. This limits the ability to compare treatments that use different prompts, settings.json, hooks, or full skill/MCP setups.

Adding `config_dir` (mapped to `CLAUDE_CONFIG_DIR`) is the cleanest approach — a single field gives per-treatment control over settings.json, hooks, skills, MCP config, and any other Claude Code configuration without adding dozens of individual fields.

## Tasks

- [x] Add `Prompt string` and `ConfigDir string` fields to the `Treatment` struct in `internal/suite/suite.go`
- [x] Update `executeSingleRun()` in `internal/executor/executor.go` to resolve prompt as `treatment.Prompt || eval.Prompt`
- [x] Update `buildRunOptions()` in `internal/executor/executor.go` to inject `CLAUDE_CONFIG_DIR` into the env map when `treatment.ConfigDir` is set
- [x] Add `prompt` and `config_dir` as supported matrix dimensions for cross-product generation
- [x] Add `prompt` and `config_dir` to defaults merging (treatment > eval > defaults precedence)
- [x] Add validation: every treatment must resolve to a non-empty prompt (from eval or treatment level)
- [x] Add validation: `config_dir` paths must exist if specified
- [x] Write unit tests for Treatment struct parsing, validation, matrix expansion, and defaults merging
- [x] Write integration test verifying prompt override and config_dir are correctly passed to the runner
- [x] Add example suite demonstrating per-treatment prompt and config_dir usage

## Acceptance Criteria

- Treatment-level `prompt` field overrides eval-level prompt when set
- Treatment-level `config_dir` field sets `CLAUDE_CONFIG_DIR` environment variable for the runner
- Both fields work in matrix expansion (can be matrix dimensions)
- Validation catches missing prompts (no eval or treatment prompt) and invalid config_dir paths
- Existing suites without these fields continue to work unchanged (backward compatible)
- Tests cover struct parsing, validation, matrix expansion, and execution
- Example suite demonstrates both features
