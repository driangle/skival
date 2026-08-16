---
id: "01m03awyn"
title: "Spike: decide and verify claude-code tool-deny enforcement mechanism"
status: completed
priority: high
effort: medium
parent: "01m03wx7s"
phase: phase-1
dependencies: []
tags: ["tool-access", "runner", "security"]
created_at: 2026-08-15
completed_at: 2026-08-16
---

# Spike: decide and verify claude-code tool-deny enforcement mechanism

> Sub-task of [[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]]. This
> is the linchpin: it decides the mechanism the other three sub-tasks build on, so
> it must land first.

## Objective

Determine — and **prove against a real agent run** — the mechanism that makes a
claude-code variant's `allowed_tools` a structural whitelist that denies unlisted
built-ins (not just MCP/custom tools). Produce a documented decision the
implementation sub-task can follow.

## Context / open questions

- Today `allowed_tools`/`disallowed_tools` flow through to `--allowedTools` /
  `--disallowedTools` (`internal/executor/runnercfg.go:39-44`), which are advisory
  for built-ins.
- **Suspected root cause of the leak:** `buildRunOptions` always adds
  `agentrunner.WithSkipPermissions()` → `--dangerously-skip-permissions`
  (`internal/executor/singlerun.go:108`), which bypasses the permission system
  entirely. Any settings-based deny rules are likely ignored while this is on.
- The external `agentrunner` claude-code runner (`v0.0.1`) exposes **no**
  `--settings` or `--permission-mode` flag — only `WorkingDir` and `Env`
  (`claudecode/runner.go`, `options.go`). So candidate mechanisms are:
  1. Write a generated `.claude/settings.json` (deny-all default + allow list) into
     the per-sample working dir (already available as `isolatedDir`) and see whether
     claude-code honours it — possibly requiring skip-permissions to be dropped or a
     `defaultMode`/permission-mode change.
  2. Extend the upstream `agentrunner` runner to pass `--settings` /
     `--permission-mode` (record what would be needed if the workdir approach is
     insufficient).

## Tasks

- [x] Reproduce the leak: run a real agent with `allowed_tools: [Read, Grep]` and
      confirm an unlisted built-in (e.g. `Bash`) currently executes
      — Experiment 2: with `--allowedTools Read Grep` (even without skip-permissions)
      Bash still executed. The allow-list is not exclusive for built-ins.
- [x] Establish how `--dangerously-skip-permissions` interacts with settings-based
      permission rules; decide whether it must be dropped/replaced for restricted
      variants — tool **removal** (`--disallowedTools` / `--tools`) is independent of
      the permission system and survives skip-permissions (Experiment 4), so
      skip-permissions can stay. Settings-file allow-lists are advisory in headless
      `-p` mode and are the wrong lever.
- [x] Prototype the deny-all + allow-listed settings config and confirm it actually
      blocks the unlisted built-in end-to-end against a real agent — Experiment 7:
      `--tools "Read,Grep"` yields `init.tools == ["Grep","Read"]` and rejects a
      Write call with *"No such tool available"*; `x.txt` never created.
- [x] If the workdir-`.claude/settings.json` approach is insufficient, document the
      exact upstream `agentrunner` change required — it is insufficient; the chosen
      lever is the `--tools` CLI flag, which the runner does not emit. Required:
      add `claudecode.WithTools(...)` → `--tools "<comma-joined>"`. See spec.
- [x] Write up the chosen mechanism (format of the generated config, where it is
      written, flags/env involved) as the design the implementation sub-task follows
      — see [tool-deny-enforcement spec](../docs/specs/tool-deny-enforcement.md).

## Decision (summary)

**Mechanism:** generate the claude CLI `--tools "<allow list>"` flag from the
variant's `allowed_tools` — an exclusive built-in whitelist (deny-by-default, no
complement to maintain, future built-ins auto-denied). **Upstream change required:**
add `claudecode.WithTools(...)` to the agentrunner claude-code runner (it emits only
`--allowedTools`/`--disallowedTools` today). `--dangerously-skip-permissions` may
stay — `--tools` acts at tool registration, independent of permissions. Full record,
evidence table, and implementer notes:
[`docs/specs/tool-deny-enforcement.md`](../docs/specs/tool-deny-enforcement.md).

## Acceptance Criteria

- A documented decision on the enforcement mechanism, grounded in an observed real
  agent run — not speculation
- Empirical proof that with the chosen mechanism, `allowed_tools: [Read, Grep]`
  blocks an unlisted built-in (built-ins, not only MCP/custom tools)
- A clear statement of any upstream `agentrunner` change required (or confirmation
  that none is needed)
