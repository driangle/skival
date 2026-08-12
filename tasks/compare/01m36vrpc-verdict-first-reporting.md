---
id: "01m36vrpc"
title: "Verdict-first reporting: better, worse, or inconclusive"
status: pending
priority: high
effort: medium
type: feature
tags: ["report", "differentiation"]
created: 2026-08-12
dependencies: ["01m3pqv5h"]
context: ["internal/report/markdown.go", "internal/report/rank.go", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "go test ./internal/report/..."
  - type: assert
    check: "A run of a suite against itself reports inconclusive rather than naming a winner"
---

# Verdict-first reporting: better, worse, or inconclusive

## Objective

The headline artifact today is a weighted composite score. It mixes correctness,
cost, duration, and quality into one number with arbitrary weights, and it is
the least defensible thing skival produces — it can rank a variant that never
executed above nothing, and it presents three decimal places of precision on top
of a single sample.

It is also the one design choice every competitor deliberately avoids. That
should be read as a signal.

Replace it as the *headline* with a per-metric verdict against the baseline:
`better`, `worse`, or `inconclusive`, each with its interval and sample count.
`inconclusive` is a first-class and frequently correct answer, and printing it
honestly is the single most differentiating thing skival can do.

## Tasks

- [ ] Add a verdict table as the primary report section: metric, delta,
      interval, n, verdict
- [ ] Demote the composite score to an opt-in section, explicitly labelled a
      heuristic rather than a measurement
- [ ] Fix pass-rate pooling while doing so: `rank.go` carefully normalizes cost
      and duration per eval, then pools pass rate globally, so an eval with 10
      samples outvotes one with 1
- [ ] Render verdicts in markdown, JSON, and HTML output
- [ ] Show sample count everywhere an interval appears

## Acceptance Criteria

- Running an unchanged configuration against itself reports `inconclusive`
- The composite score no longer appears unless requested
- Pass rate is aggregated per eval, consistent with cost and duration
