---
id: "01kz6mm74"
title: "Reconcile 'statistical confidence' claims with what is computed"
status: completed
priority: medium
effort: medium
type: improvement
tags: ["report", "stats", "docs"]
created: 2026-08-04
completed_at: 2026-08-05
---

# Reconcile 'statistical confidence' claims with what is computed

## Objective

The README promises comparing treatments "head-to-head with statistical rigor"
and running multiple samples "for statistical confidence." What
`internal/result/aggregate.go` actually computes is descriptive statistics only:
median, min, max, and coefficient of variation (CV only at n >= 3). There is no
confidence interval, standard error, or significance test — so a user cannot
tell whether a treatment beating control is signal or noise, which is the entire
point of a controlled comparison.

Pick one of two directions (or both): make the claim true, or soften it.

## Tasks

- [x] Decide scope: add inferential statistics vs. reword the claims
      — chose to soften/reword the claims (descriptive stats only)
- [ ] (If adding) compute a confidence interval per metric (e.g. bootstrap or
      t-interval) and surface it in the aggregate + reports — n/a (softening)
- [ ] (If adding) add a control-vs-treatment comparison (e.g. bootstrap
      difference CI, or Mann-Whitney/Welch) and flag differences that clear it
      — n/a (softening)
- [x] (If softening) update README/docs to say "descriptive statistics
      (median, spread, CV)" and drop "rigor"/"confidence" wording
- [x] Reconsider the CV `n >= 3` cutoff and document why small-n metrics are nil
      — documented rationale in `cv()` and getting-started docs; kept n>=3
- [ ] Tests for any new statistic (`internal/result/aggregate_test.go`)
      — n/a, no new statistic added

## Acceptance Criteria

- Documentation and computed output agree: either real inferential stats exist,
  or the marketing language no longer implies them
- Any added statistic has unit tests and appears in markdown + JSON reports
