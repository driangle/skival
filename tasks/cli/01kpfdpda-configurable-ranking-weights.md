---
title: "Configurable ranking weights"
id: "01kpfdpda"
status: in-progress
priority: medium
type: feature
tags: ["reporting", "configuration"]
created: "2026-04-18"
---

# Configurable ranking weights

## Objective

Make the ranking weights for the composite treatment score configurable via suite.yaml instead of hardcoded Go constants. Currently weights are fixed at 60% correctness, 28% cost, 12% duration. Different use cases prioritize different metrics — cost-sensitive users want cost weighted higher, latency-sensitive users want duration weighted higher.

### Configuration

```yaml
ranking:
  weights:
    correctness: 0.60    # default: 0.60
    cost: 0.28           # default: 0.28
    duration: 0.12       # default: 0.12
```

Weights must sum to 1.0. When omitted, current defaults apply.

## Tasks

- [ ] Add `Ranking` struct with `Weights` to suite schema (correctness, cost, duration floats)
- [ ] Add validation: all weights >= 0, weights sum to 1.0 (with small epsilon tolerance)
- [ ] Update `RankTreatments()` in `report/rank.go` to accept weights as a parameter instead of using constants
- [ ] Keep current constants as defaults when no ranking config is provided
- [ ] Thread ranking config from loaded suite through to report generation
- [ ] Add suite loader tests for ranking config parsing and validation
- [ ] Add ranking tests with custom weights
- [ ] Update documentation (configuration.md) with ranking config

## Acceptance Criteria

- Setting custom weights in suite.yaml changes the composite score calculation
- Omitting ranking config produces identical results to current behavior
- Validation rejects weights that don't sum to 1.0
- Validation rejects negative weights
- Weights are passed through the full pipeline: suite load → report generation
