---
id: "01m3qhtgp"
title: "--until-decided adaptive sampling with a cost budget"
status: pending
priority: medium
effort: large
type: feature
tags: ["cli", "executor", "differentiation"]
created: 2026-08-12
dependencies: ["01m3pqv5h"]
context: ["apps/cli/cmd/run.go", "internal/executor/executor.go", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "go test ./internal/executor/... ./apps/cli/..."
  - type: assert
    check: "Sampling stops early when the interval resolves, and stops at the budget cap otherwise, reporting which occurred"
---

# --until-decided adaptive sampling with a cost budget

## Objective

Users under-sample because agent runs cost money, and a fixed `--samples N` is
either wasteful (the answer was obvious after 3) or useless (the answer needed
20). Adaptive sampling resolves both: keep drawing paired samples until the
delta interval excludes zero, or until a cost or wall-clock budget is spent —
then report which of the two happened.

No competitor does this. It converts skival's biggest liability (expensive,
noisy runs) into the reason to use it.

## Tasks

- [ ] Add `--until-decided` with `--max-cost`, `--max-samples`, and
      `--max-duration` bounds
- [ ] Re-evaluate the interval after each paired round rather than after each
      individual run
- [ ] Stop on resolution or budget exhaustion and record the stopping reason in
      results and reports
- [ ] Guard against optional-stopping bias — document that early stopping
      inflates apparent effects, and require a minimum round count
- [ ] Emit progress showing samples drawn, budget consumed, and current interval

## Acceptance Criteria

- A clearly separated pair stops early and reports `decided`
- An indistinguishable pair exhausts the budget and reports `inconclusive
  (budget exhausted)`
- Stopping reason is persisted and appears in reports
