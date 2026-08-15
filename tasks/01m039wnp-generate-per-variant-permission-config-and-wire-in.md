---
id: "01m039wnp"
title: "Generate per-variant permission config and wire into claude-code runner"
status: pending
priority: high
effort: large
parent: "01m03wx7s"
phase: phase-1
dependencies: ["01m03awyn"]
tags: ["tool-access", "runner", "security"]
created_at: 2026-08-15
---

# Generate per-variant permission config and wire into claude-code runner

> Sub-task of [[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]].
> Depends on the mechanism decided in
> [[01m03awyn-spike-decide-and-verify-claude-code-tool-deny-enfo]].

## Objective

Implement generation of the deny-all + allow-listed permission/settings config from
a variant's declared `allowed_tools`, write it into the per-sample working
directory, and wire it into the claude-code invocation — replacing reliance on the
advisory `--allowedTools`/`--disallowedTools` flags for built-ins.

## Design notes

- Keep it runner-scoped: dispatch already happens in `buildRunnerSpecificOpts`
  (`internal/executor/runnercfg.go:14-32`); add the plumbing in
  `buildClaudeCodeOpts` and/or `buildRunOptions` (`internal/executor/singlerun.go`).
- Generate the config from the variant's `allowed_tools` plus whatever the
  suite/CLI provides — **do not** hardcode a list of tools that exist (project
  convention: skival must not assume the user's stack/tool set).
- Write the generated config into the sample's isolated working dir
  (`isolatedDir` / `resolveWorkdir`, `internal/executor/run.go`) or a temp path, and
  point the runner at it via the mechanism chosen in the spike.
- Reconcile the always-on `agentrunner.WithSkipPermissions()`
  (`internal/executor/singlerun.go:108`) with the enforcement mechanism per the
  spike's decision.

## Tasks

- [ ] Add a config generator that turns `allowed_tools` into the deny-all +
      allow-listed permission/settings structure decided in the spike
- [ ] Write the generated config into the per-sample working dir (respecting
      existing isolation) and wire the runner to consume it
- [ ] Reconcile `--dangerously-skip-permissions` with the enforcement mechanism
- [ ] Unit tests for config generation from `allowed_tools` (including empty/unset)

## Acceptance Criteria

- A variant declaring `allowed_tools: [Read, Grep]` produces a generated config
  that, when consumed by the runner, denies every other tool — including
  newly-added built-ins — without editing any deny list
- Enforcement is generated from the allow list, not a hardcoded complement
- Config generation is unit-tested; existing suites relying on `allowed_tools`
  continue to run
