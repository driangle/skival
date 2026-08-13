---
title: "Ranking overstates confidence at small sample sizes"
id: "01kzyyvmv"
status: pending
priority: medium
type: feature
tags: ["ranking", "statistics", "reporting"]
created: "2026-08-13"
phase: phase-3
---

# Ranking overstates confidence at small sample sizes

## Objective

The ranking reads more confident than an n=3 (or n=9) run actually supports.
In one run, `plugin-skill 0.915` vs `no-skill 0.746` came down to a single
sample out of nine, yet the composite score presents that coin-flip-scale
difference with three decimal places.

CV is computed for cost and duration but nothing is computed for correctness,
so there's no signal that a correctness difference is within noise. Adding a
confidence measure on pass rate — and a "not significant" flag when variants'
intervals overlap — would stop people (the reporter included) from over-reading
a run.

## Tasks

- [ ] Compute a confidence interval on pass rate per variant (Wilson interval on the pass/fail counts).
- [ ] Surface the interval (or its width) alongside the correctness score in the report, so the precision of the point estimate is visible.
- [ ] Flag pairs/rankings as "not significant at this sample size" when variants' pass-rate intervals overlap.
- [ ] Consider whether the composite score's displayed precision should scale with sample size (avoid three decimals on a nine-sample run).
- [ ] Add tests: Wilson interval computation, overlap/"not significant" flagging, and rendering in the report.

## Acceptance Criteria

- The report includes a confidence interval (Wilson) on pass rate for each variant.
- When two variants' pass-rate intervals overlap, the report flags the difference as not significant at the current sample size.
- Tests cover the interval math and the significance flagging, including the overlapping-intervals case.
