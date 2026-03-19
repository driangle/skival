---
title: "Weighted ranking"
id: "01km2hm2y"
status: completed
priority: medium
type: feature
tags: ["phase-3", "reporting"]
dependencies: ["01km2hm2h"]
created: "2026-03-19"
phase: phase-3
---

# Weighted ranking

## Objective

Rank treatments by a weighted composite score: 60% pass rate, 28% cost (lower is better), 12% duration (lower is better). Normalize scores so the best performer in each dimension gets 1.0.

## Tasks

- [ ] Implement ranking logic in `internal/report/rank.go`
- [ ] Compute per-treatment stats across all evals: pass rate, median cost, median duration
- [ ] Normalize each dimension: best = 1.0, worst = 0.0 (or inversely for cost/duration)
- [ ] Compute weighted composite: 0.6*pass + 0.28*cost_norm + 0.12*duration_norm
- [ ] Sort treatments by composite score descending
- [ ] Unit tests with known inputs to verify ranking output

## Acceptance Criteria

- Treatment with highest pass rate, lowest cost, and fastest time ranks #1
- Normalization handles ties and single-treatment cases
- Weights sum to 1.0
- Rankings are deterministic given the same inputs
