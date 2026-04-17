---
title: "Wire default registry in CLI run command"
id: "01kpcvjsv"
status: pending
priority: critical
type: feature
effort: small
tags: ["backend", "cli"]
dependencies: ["01kpcvjkv"]
created: "2026-04-17"
---

# Wire default registry in CLI run command

## Objective

Replace the hardcoded `claudecode.NewRunner()` in the CLI run command with a `defaultRegistry()` that registers all built-in runners, and pass it to the executor.

## Tasks

- [ ] Create `defaultRegistry()` function in `apps/cli/cmd/run.go` that registers `claude-code` and `ollama` factories
- [ ] Replace the single `runner` creation with `defaultRegistry()` call
- [ ] Pass the registry to `executor.Execute` instead of the single runner
- [ ] Ensure the logger is wired into the claude-code factory

## Acceptance Criteria

- [ ] `skival run` with a suite that has no `runner` field works as before (defaults to claude-code)
- [ ] `skival run` with a suite specifying `runner: ollama` on a treatment creates an ollama runner
- [ ] The claude-code runner receives the configured logger
- [ ] Existing CLI behavior is unchanged for single-runner suites
