---
id: "01m3pqv5h"
title: "Bootstrap confidence intervals on per-metric deltas"
status: pending
priority: high
effort: medium
type: feature
tags: ["result", "statistics", "differentiation"]
created: 2026-08-12
dependencies: ["01m3wmqcn"]
context: ["internal/result/aggregate.go", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "go test ./internal/result/..."
  - type: assert
    check: "Two identical distributions produce an interval spanning zero; a large separation produces one that excludes zero"
---

# Bootstrap confidence intervals on per-metric deltas

## Objective

skival reports medians, min/max, and a coefficient of variation, then presents a
single composite score with three decimal places. Nothing in the output tells a
user whether a difference between two variants is larger than run-to-run noise —
which, for agent runs, it usually is not.

Bootstrap percentile intervals on the *delta versus baseline* are the right
tool: they need no distributional assumption, they are straightforward to
implement in the standard library, and they are easy to explain honestly. This
is deliberately not significance testing — no p-values, no null hypothesis
framing, no claim the sample sizes cannot support.

## Tasks

- [ ] Implement a seeded bootstrap resampler over paired samples
- [ ] Compute percentile intervals for the pass-rate, cost, and duration deltas
      against the baseline variant
- [ ] Make the seed configurable so a report is reproducible from saved results
- [ ] Refuse to emit an interval below a minimum sample count, and say so rather
      than emitting a wide one
- [ ] Persist intervals alongside aggregates so `report` does not recompute them
      differently

## Acceptance Criteria

- Identical inputs produce an interval spanning zero
- A clear separation produces an interval excluding zero
- Intervals are reproducible given the same seed and inputs
