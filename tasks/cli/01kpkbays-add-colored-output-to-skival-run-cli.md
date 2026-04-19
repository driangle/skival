---
title: "Add colored output to skival run CLI"
id: "01kpkbays"
status: completed
priority: medium
type: feature
tags: ["cli", "ux"]
created: "2026-04-19"
completed_at: 2026-04-19
---

# Add colored output to skival run CLI

## Objective

Add color to `skival run` CLI output to make status information (PASS/FAIL/ERROR, progress, costs) easier to scan at a glance. Currently all output is plain text with no color differentiation.

## Tasks

- [x] **Design the color scheme**: Survey the output in `internal/executor/progress.go` and `internal/executor/printer.go`. Propose a color mapping (e.g. green for PASS/ok/done, red for FAIL/ERROR, cyan for labels, dim for elapsed time/costs). Write up the scheme with before/after examples.
- [x] **Confirm design with user**: Present the proposed color scheme and get approval before implementing.
- [x] **Choose coloring approach**: Decide between `fatih/color` (handles NO_COLOR and non-TTY automatically) or raw ANSI codes (no new dependency, consistent with existing cursor control). Document the decision.
- [x] **Implement colored output in progress.go**: Add color to real-time progress lines — PASS/FAIL/ERROR labels, eval headers, elapsed time, cost.
- [x] **Implement colored output in printer.go**: Add color to the results summary table — status column, totals.
- [x] **Implement colored output in validate.go**: Color the OK/FAIL validation output.
- [x] **Respect NO_COLOR and non-TTY**: Ensure colors are suppressed when `NO_COLOR` env var is set or output is not a terminal (piped/redirected).
- [x] **Add tests**: Verify colored output renders correctly and that NO_COLOR / non-TTY suppression works.

## Acceptance Criteria

- PASS/ok/done render in green, FAIL/ERROR in red, warnings in yellow, labels in cyan, secondary info (elapsed time, costs) in dim/gray
- Colors are automatically disabled when stdout/stderr is not a TTY or when `NO_COLOR` environment variable is set
- Existing non-color ANSI cursor control (carriage return, erase-to-EOL) continues to work correctly
- No visual regressions in piped/redirected output (plain text, no escape codes)
