---
id: "01kz6mm74"
title: "Reconcile 'statistical confidence' claims with what is computed"
status: pending
priority: medium
effort: medium
type: improvement
tags: ["report", "stats", "docs"]
created: 2026-08-04
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

- [ ] Decide scope: add inferential statistics vs. reword the claims
- [ ] (If adding) compute a confidence interval per metric (e.g. bootstrap or
      t-interval) and surface it in the aggregate + reports
- [ ] (If adding) add a control-vs-treatment comparison (e.g. bootstrap
      difference CI, or Mann-Whitney/Welch) and flag differences that clear it
- [ ] (If softening) update README/docs to say "descriptive statistics
      (median, spread, CV)" and drop "rigor"/"confidence" wording
- [ ] Reconsider the CV `n >= 3` cutoff and document why small-n metrics are nil
- [ ] Tests for any new statistic (`internal/result/aggregate_test.go`)

## Acceptance Criteria

- Documentation and computed output agree: either real inferential stats exist,
  or the marketing language no longer implies them
- Any added statistic has unit tests and appears in markdown + JSON reports
