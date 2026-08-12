---
id: "01m3wmqcn"
title: "Paired, interleaved execution of baseline and variants"
status: pending
priority: high
effort: medium
type: feature
tags: ["executor", "measurement", "differentiation"]
created: 2026-08-12
dependencies: ["01m3pbbc2"]
context: ["internal/executor/executor.go", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "go test ./internal/executor/..."
  - type: assert
    check: "With samples=3 and two variants, execution order alternates baseline/variant rather than running all baseline samples first"
---

# Paired, interleaved execution of baseline and variants

## Objective

`executeEval` runs variants one at a time, each completing all of its samples
before the next variant starts. A provider slowdown, a load spike, or a silent
model update during the run therefore hits one arm and not the other, and shows
up as a difference between variants.

Paired execution — alternating arms sample by sample — makes drift a shared
nuisance rather than a confound, and is the foundation for every uncertainty
estimate in this phase. It is also a structural advantage: competitors model
evaluation as a matrix sweep, not a paired trial, and cannot retrofit this
cheaply.

## Tasks

- [ ] Restructure eval execution so sample *i* of every variant runs before
      sample *i+1* of any variant
- [ ] Preserve per-sample pairing identity in results, so downstream analysis can
      compare like with like rather than pooling
- [ ] Keep the reset hook semantics correct under interleaving (reset runs
      between arms, not only between variants)
- [ ] Make interleaving the default; keep variant-at-a-time available for suites
      whose setup hooks cannot support it
- [ ] Document the interaction with `--parallel-variants`, which currently shares
      one working directory and skips the reset hook

## Acceptance Criteria

- Execution order alternates arms within an eval
- Results record which samples were paired
- Existing hook-ordering tests still pass
