---
title: "Multi-sample aggregation"
id: "01km2hm2h"
status: completed
priority: medium
type: feature
tags: ["phase-3", "reporting"]
dependencies: ["01km2hkxn"]
created: "2026-03-19"
phase: phase-3
---

# Multi-sample aggregation

## Objective

When --samples N > 1, run each treatment N times and aggregate results: median cost/duration, conservative pass logic (all must pass), and variance statistics (min, max, CV).

## Tasks

- [ ] Update execution flow to loop N times per treatment
- [ ] Implement aggregation in `internal/result/aggregate.go`: median, min, max, CV computation
- [ ] Conservative pass: any fail → fail, any nil → nil, all pass → pass
- [ ] Only compute CV for 3+ samples
- [ ] Store individual run metrics alongside aggregate in TreatmentResult
- [ ] Unit tests for aggregation math (median of odd/even counts, CV calculation)

## Acceptance Criteria

- `--samples 3` runs each treatment 3 times
- Aggregated cost/duration are medians, not means
- Pass is false if any individual run failed
- CV is reported for 3+ samples, omitted for fewer
- Individual run details are preserved for drill-down
