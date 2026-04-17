---
title: "Multi-dimensional reporting and analysis"
id: "01kpcv08d"
status: pending
priority: medium
type: feature
tags: ["reporting", "analysis"]
created: "2026-04-17"
---

# Multi-dimensional reporting and analysis

## Objective

Extend reporting to support analysis across multiple dimensions — slicing results by runner, model, skillset, or any matrix dimension. Currently rankings are flat (treatment-level only) with hardcoded weights (60/28/12). There's no way to answer "which model wins regardless of runner?" or compare results across suite runs.

## Tasks

- [ ] Add dimension metadata to `TreatmentResult` so reports know which values each treatment used
- [ ] Implement per-dimension aggregation in `internal/report/rank.go` — group by one dimension, average across others
- [ ] Add `--group-by` flag to `skival report` command (e.g., `--group-by runner`, `--group-by model`)
- [ ] Make ranking weights configurable via suite.yaml `defaults.weights: { pass: 0.6, cost: 0.28, duration: 0.12 }`
- [ ] Add cross-suite comparison: `skival compare <results-dir-1> <results-dir-2>` command
- [ ] Add statistical significance testing (e.g., Mann-Whitney U) between treatment pairs when samples >= 5
- [ ] Update markdown report to include per-dimension summary tables when matrix is used
- [ ] Add tests for dimension-grouped ranking and configurable weights

## Acceptance Criteria

- `skival report --group-by runner` groups treatments by runner and shows aggregated metrics per runner
- Custom ranking weights in suite.yaml are respected in composite score calculation
- `skival compare` shows side-by-side results from two different runs with delta columns
- Statistical significance is computed and displayed when sample count is sufficient
- Default behavior (no `--group-by`, default weights) is unchanged from current
