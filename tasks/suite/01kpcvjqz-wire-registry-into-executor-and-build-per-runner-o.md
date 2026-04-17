---
title: "Wire registry into executor and build per-runner options"
id: "01kpcvjqz"
status: pending
priority: critical
type: feature
effort: medium
tags: ["backend", "executor"]
dependencies: ["01kpcvjj5", "01kpcvjkv"]
created: "2026-04-17"
---

# Wire registry into executor and build per-runner options

## Objective

Change the executor to accept a `*registry.Registry` instead of a single `agentrunner.Runner`, resolve the runner per treatment at execution time, and translate `runner_config` into runner-specific `agentrunner.Option` values.

## Tasks

- [ ] Change `Execute` signature to accept `*registry.Registry` instead of `agentrunner.Runner`
- [ ] Add a runner cache (`map[string]agentrunner.Runner`) in `Execute` so the same runner type is reused across treatments
- [ ] In `executeTreatment`, resolve the runner from the registry (defaulting to `"claude-code"` when empty)
- [ ] Extract `buildRunnerSpecificOpts(runner string, config map[string]any) []agentrunner.Option` that dispatches to per-runner option builders
- [ ] Implement `buildClaudeCodeOpts(config)` mapping: `allowed_tools`, `disallowed_tools`, `mcp_config`, `max_budget_usd`
- [ ] Implement `buildOllamaOpts(config)` mapping: `temperature`, `num_ctx`, `num_predict`, `top_p`, `top_k`, `seed`, `stop`, `think`
- [ ] Remove the old `AllowedTools` handling from `buildRunOptions` (now handled via `runner_config`)
- [ ] Update existing executor tests to pass a registry instead of a single runner

## Acceptance Criteria

- [ ] `Execute` accepts a registry and resolves runners per treatment
- [ ] Runners are cached by name — creating the same runner type twice reuses the instance
- [ ] `runner_config` keys are correctly mapped to agentrunner options for claude-code and ollama
- [ ] Unknown `runner_config` keys produce a log warning (not an error)
- [ ] All existing executor tests pass with the new signature
- [ ] New unit tests cover per-runner option building for both claude-code and ollama configs
