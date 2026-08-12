---
id: "01m3xt9pn"
title: "Make persisted results self-describing (weights and token usage)"
status: pending
priority: high
effort: small
type: bug
tags: ["persist", "report"]
created: 2026-08-12
context: ["internal/persist/persist.go", "internal/persist/load.go", "apps/cli/cmd/report.go"]
verify:
  - type: bash
    run: "go test ./internal/persist/... ./internal/report/..."
  - type: assert
    check: "skival report on a saved results dir reproduces the rankings in that dir's summary.md"
---

# Make persisted results self-describing (weights and token usage)

## Objective

Two fields are collected and then lost.

**Ranking weights.** `persist.Save` takes weights and writes the resulting
rankings, but not the weights themselves. `apps/cli/cmd/report.go:24` then
hardcodes `report.DefaultWeights()`. On the results directory committed in this
repo, the same data yields two different answers:

| | concise | detailed |
| --- | --- | --- |
| `summary.md`, written by `run` | 0.960 | 0.808 |
| `skival report` on that directory | 1.000 | 0.860 |

The re-report also prints a QUALITY column that contributes nothing, because the
quality weight silently reset to 0.

**Token usage.** `RunResult.Usage` is populated and never appears in any report
or in the persisted `run-N.json`, despite being the second item in README's
headline sentence.

## Tasks

- [ ] Persist the resolved ranking weights in `summary.json` and have
      `persist.Load` return them
- [ ] Have `skival report` use the persisted weights, falling back to defaults
      only when absent (older results)
- [ ] Persist `usage` in `run-N.json` and surface token counts in the markdown,
      JSON, and HTML reports
- [ ] Round-trip the eval `Name` so `report` does not print IDs where `run`
      printed names

## Acceptance Criteria

- `skival report <dir>` reproduces the rankings in that directory's own
  `summary.md`
- Token usage appears in reports and persisted results
