---
title: "Pre-flight tool-leak detection from system/init event"
id: "01m030n6n"
status: completed
priority: high
type: feature
tags: ["tool-access", "observability", "runner"]
created: "2026-08-15"
effort: medium
completed_at: 2026-08-15
---

# Pre-flight tool-leak detection from system/init event

## Objective

Catch tool-access leakage on the **first sample** and warn before spending the rest
of the budget — the cheapest available win.

When an agent starts, the stream's `system` / `init` event enumerates the tools it
actually has. skival can diff that list against the variant's declared
`allowed_tools` on the first sample and warn immediately, e.g.:

```
⚠ variant "no-skill": 6 tools available beyond allowed_tools:
  Skill, TaskCreate, ToolSearch, …
```

This catches leaks (built-ins that outlive the advisory flags, or tools that simply
don't get denied) with **zero maintenance** — it reads the ground truth the agent
reports about itself rather than a list skival has to keep current. In the motivating
run it would have fired at sample 1 instead of after ~$75 of a 4-variant suite that
was silently measuring one configuration.

**Why this matters beyond a single bug:** the failure was undetectable from skival's
own output. A benchmark that can't show you what the agent had access to can't be
trusted, however correct its enforcement is. This task and the tool census
([[01m036fad-per-variant-tool-census-in-the-report]]) are the observability half of
the fix; the enforcement half is
[[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]].

## Design notes

- The raw `system`/`init` line is already available: `collectConversation` keeps
  every `msg.Raw` into `RunResult.Conversation`
  (`internal/executor/singlerun.go:46-53`, `internal/result/result.go:25`).
  The init envelope is `type:"system", subtype:"init"` with a `tools` array (see
  external `claudecode/types.go:30`, detected at `claudecode/runner.go:248`).
  skival has **no parser** for it today — that's the net-new piece.
- Parse the first sample's init event, extract the `tools` array, and diff against
  the variant's resolved `allowed_tools`. Emit a warning (not a hard failure — the
  hard-fail path is the backstop verifier
  [[01m037pwn-tool-not-used-verifier-as-a-leakage-backstop]]).
- **Consider reusing vibeview** to inspect the session and pull the available-tools
  list rather than hand-rolling JSONL parsing — skival itself (or the client) can
  drive vibeview over the session. Evaluate that before building a bespoke parser.
- Don't assume a fixed built-in set — the whole point is to read what the agent
  reports. Handle runners whose init event has no `tools` field by skipping quietly.
- Fire once per variant (first sample), not every sample, to avoid noise.

## Tasks

- [x] Add a parser for the `type:"system", subtype:"init"` message that extracts the
      `tools` array from `RunResult.Conversation` (`internal/toolaudit/toolaudit.go`)
- [x] On the first sample of each variant, diff available tools vs resolved
      `allowed_tools` and emit a warning listing the extras
      (`internal/executor/preflight.go`, wired in `samples.go`, warning in `progress.go`)
- [x] Evaluate reusing vibeview for session inspection vs. a bespoke parser; note the
      decision in the task/worklog — **bespoke parser chosen**: the init line is
      already in memory as `RunResult.Conversation`, so vibeview would mean persisting
      the session and shelling out to re-read data we already hold. A ~20-line
      dependency-free parser is simpler and needs no `--link-sessions`.
- [x] Gracefully no-op when the runner provides no init tool list
      (`AvailableTools` returns `ok=false`)
- [x] Tests: init-event parsing, the diff/warning logic, and the no-`tools` case
      (`internal/toolaudit/toolaudit_test.go`, `internal/executor/preflight_test.go`)

## Acceptance Criteria

- Running a suite where the agent has tools beyond `allowed_tools` prints a clear
  warning naming the extra tools, on the first sample, before the bulk of the budget
  is spent
- No warning fires when available tools ⊆ `allowed_tools`
- Runners that don't emit a tool list in their init event are handled without error
- No hand-maintained list of built-in tool names is introduced
