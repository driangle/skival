---
id: "01m3x4yen"
title: "Report wall-clock duration from the exec runner"
status: pending
priority: high
effort: small
type: bug
tags: ["exec", "measurement"]
created: 2026-08-12
context: ["internal/runners/exec/exec.go"]
verify:
  - type: bash
    run: "go test ./internal/runners/exec/..."
  - type: assert
    check: "A command that sleeps 1s reports a duration of roughly 1000ms, not 0ms"
---

# Report wall-clock duration from the exec runner

## Objective

`buildResult` (`internal/runners/exec/exec.go:138`) sets `Text`, `ExitCode`,
`IsError`, and cost/usage from the final event — but never `Duration`. Every
exec run reports `0ms`. Verified: a script that sleeps for one second reports
`0ms`.

"Measures **time to completion**" is the first claim in README.md, and it is
broken for the one runner that makes skival stack-agnostic — the runner the
differentiation strategy is built on. Duration is also 12% of the composite, so
that dimension is degenerate for every exec suite: all variants tie at 1.0.

## Tasks

- [ ] Measure wall clock around process start/wait and set `Result.Duration`
- [ ] Prefer a duration reported by the final JSONL event when present, falling
      back to wall clock
- [ ] Add a test asserting a sleeping command reports a non-zero duration

## Acceptance Criteria

- An exec variant running a 1s command reports ~1000ms
- Duration ranking is no longer degenerate across exec variants
