---
id: "01m039wnp"
title: "Generate per-variant permission config and wire into claude-code runner"
status: completed
priority: high
effort: large
parent: "01m03wx7s"
phase: phase-1
dependencies: ["01m03awyn"]
tags: ["tool-access", "runner", "security"]
created_at: 2026-08-15
completed_at: 2026-08-16
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

- [x] Add a config generator that turns `allowed_tools` into the deny-all +
      allow-listed structure decided in the spike — `toolaudit.BuiltinWhitelist`
      derives the exclusive `--tools` value from `allowed_tools` (base names,
      MCP-excluded, deduped; empty/only-MCP → `[""]` to disable all built-ins).
- [x] Wire the runner to consume it — `buildClaudeCodeOpts` maps `allowed_tools`
      to `claudecode.WithTools(...)` (v0.0.2). **No file is written** into the
      workdir: the spike chose the CLI `--tools` flag over a `.claude/settings.json`
      (settings-file allow-lists are advisory in headless `-p` mode), so per-sample
      isolation is unaffected.
- [x] Reconcile `--dangerously-skip-permissions` with the enforcement mechanism —
      `--tools` acts at tool registration, independent of the permission system, so
      it composes with skip-permissions (v0.0.2 `buildArgs` keeps both). No change
      needed; skip-permissions stays.
- [x] Unit tests for config generation from `allowed_tools` (including empty/unset)
      — `TestBuiltinWhitelist` (toolaudit) + `TestBuildClaudeCodeOptsToolsWhitelist`
      and extended `TestBuildClaudeCodeOpts` (executor).

## Notes

- Depends on agentrunner **v0.0.2** (`WithTools`); `go.mod` bumped. The v0.0.2 rename
  `WithSkipPermissions` → `WithDangerouslySkipPermissions` was applied across
  `singlerun.go`, `verifier/judge.go`, `verifier/comparative.go`.
- Requires `claude` CLI **≥ 2.1.0** (runner's `MinCLIVersion`, first release shipping
  `--tools`). Called out for the e2e/docs sub-task `01m03xmkx`.
- **MCP boundary:** `--tools` scopes built-ins only; MCP tools (`mcp__*`) are governed
  by `mcp_config`. Empirical MCP-interaction verification is left to `01m03xmkx` (no
  current suite mixes MCP entries into `allowed_tools`).

## Acceptance Criteria

- A variant declaring `allowed_tools: [Read, Grep]` produces a generated config
  that, when consumed by the runner, denies every other tool — including
  newly-added built-ins — without editing any deny list
- Enforcement is generated from the allow list, not a hardcoded complement
- Config generation is unit-tested; existing suites relying on `allowed_tools`
  continue to run
