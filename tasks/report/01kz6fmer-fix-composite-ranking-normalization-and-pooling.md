---
id: "01kz6fmer"
title: "Fix composite ranking normalization and cross-eval pooling"
status: completed
priority: high
effort: medium
type: bug
tags: ["report", "ranking"]
created: 2026-08-04
verify:
  - type: bash
    run: "go test ./internal/report/..."
  - type: assert
    check: "For a control + 1 treatment run, a small vs large metric gap produces different composite scores (not just 0/1)"
completed_at: 2026-08-05
---

# Fix composite ranking normalization and cross-eval pooling

## Objective

Two problems in `internal/report/rank.go` make the composite score misleading,
especially in the flagship control + 1 treatment case.

1. **Min-max normalization degenerates.** `normalize` scales each metric to
   `[0,1]` across variants. With exactly two variants every metric is either
   `0` or `1`, so the "weighted composite score" collapses into a weighted
   coin-flip of *who won each axis* with zero sensitivity to magnitude
   (winning cost by 1% and by 90% score identically).
2. **Cross-eval pooling.** `collectStats` appends every run's cost/duration
   from *all evals* into one slice per variant, then takes a single median —
   mixing the distributions of a cheap task and an expensive task into one
   number.

## Tasks

- [x] Normalize per-eval, then aggregate across evals (e.g. mean of per-eval
      normalized scores), instead of pooling raw costs/durations globally
- [x] Replace or supplement min-max with a magnitude-sensitive scheme
      (e.g. ratio-to-best or ratio-to-control) so a 1% gap and a 90% gap differ
- [x] Decide and document how a single-variant eval is scored (currently
      `normLowerBetter`/`normHigherBetter` return 1.0 when all values are equal)
- [x] Update `rank_test.go` with a 2-variant case asserting magnitude sensitivity
      and a multi-eval case asserting per-eval aggregation

## Acceptance Criteria

- Composite scores reflect the *size* of metric differences, not just ordering
- Per-eval costs/durations are not pooled into a single global median
- `go test ./internal/report/...` passes
