---
title: "Add cost preview (--dry-run) and suite-level budget cap (--max-cost)"
id: "01kzyddca"
status: pending
priority: medium
type: feature
tags: ["cost", "budget", "cli", "ux"]
created: "2026-08-13"
phase: phase-4
---

# Add cost preview (--dry-run) and suite-level budget cap (--max-cost)

## Objective

There's no cost preview and no suite-level budget cap. Before committing to a
run, the only way to estimate cost is by hand (e.g. "27 samples × ~$0.25 ≈ $7").
Two additions would fix this:

1. **`--dry-run` cost preview**: print the run matrix — evals × variants ×
   samples, and the resolved model per variant — so the total sample count and
   configuration are visible in one command. Optionally feed prior medians from
   `--results-dir` into the estimate to make it accurate rather than a
   hand-guess.

2. **`--max-cost` circuit breaker**: `max_budget_usd` already exists
   per-variant in `runner_config`, but there's no suite-level cap like
   `skival run --max-cost 10` to abort the whole run once cumulative spend
   crosses a threshold.

## Tasks

- [ ] Add `--dry-run` that prints the resolved run matrix (evals × variants × samples) and the resolved model per variant, then exits without running.
- [ ] When `--results-dir` is provided, use prior per-eval/variant medians to produce an estimated total cost in the dry-run output.
- [ ] Add a suite-level `skival run --max-cost <usd>` circuit breaker that aborts the run once cumulative spend exceeds the cap.
- [ ] Decide and document abort semantics (stop before starting the next sample vs. mid-sample) and what gets reported on abort.
- [ ] Add tests for the dry-run matrix output, the estimate computation from a results dir, and the max-cost abort behavior.

## Acceptance Criteria

- `skival run --dry-run` prints the full run matrix and resolved model per variant without executing any samples.
- With `--results-dir`, the dry-run output includes an estimated total cost derived from prior medians.
- `skival run --max-cost <usd>` aborts the run once cumulative spend exceeds the cap, with a clear message and non-zero exit.
- Tests cover the dry-run output, the estimate, and the max-cost abort.
