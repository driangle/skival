---
title: "Verifier pipeline composition"
id: "01km2hm1r"
status: pending
priority: medium
type: feature
tags: ["phase-2", "verifier"]
dependencies: ["01km2hkyh", "01km2hkzt", "01km2hm0t"]
created: "2026-03-19"
phase: phase-2
---

# Verifier pipeline composition

## Objective

Compose individual verifiers into a pipeline that runs in order (execute → output match → state assertions → script) based on the eval's correctness config, short-circuiting on first failure.

## Tasks

- [ ] Implement pipeline builder that reads correctness config and assembles the appropriate verifiers
- [ ] Run verifiers in sequence: execute → output → state → script
- [ ] Short-circuit on first failure, returning that verifier's result
- [ ] Skip verifiers whose config fields are absent/empty
- [ ] Integrate pipeline into the execution flow so each run is verified
- [ ] Return composite VerifyResult with which steps passed/failed

## Acceptance Criteria

- Pipeline only includes verifiers relevant to the eval's correctness config
- First failure stops the pipeline and returns the failure reason
- When all verifiers pass, the overall result is pass
- Pipeline integrates cleanly into the execution orchestrator from the result types task
