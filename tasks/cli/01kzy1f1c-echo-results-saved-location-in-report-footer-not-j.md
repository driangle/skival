---
title: "Echo results-saved location in report footer, not just stderr"
id: "01kzy1f1c"
status: completed
priority: low
type: chore
tags: ["reporting", "ux"]
created: "2026-08-13"
phase: phase-3
completed_at: 2026-08-13
---

# Echo results-saved location in report footer, not just stderr

## Description

The `Results saved to ...` line goes to stderr. That's reasonable on its own,
but a reader who redirects stderr to a log file briefly loses track of where
the run landed — the results-dir path is nowhere in the report itself.

Echo the results-saved location in the report footer too (in addition to the
existing stderr line), so the path is recoverable from the report regardless of
stream redirection.

## Tasks

- [x] Add the results-dir / results-saved location to the report footer.
- [x] Keep the existing stderr line as-is (this is additive, not a move).
- [x] Update report tests/snapshots to include the footer path.

## Acceptance Criteria

- The report footer includes the location where results were saved.
- The stderr `Results saved to ...` line still prints.
- Report tests/snapshots reflect the footer path.
