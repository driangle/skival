---
title: "Result comparison across runs"
id: "01kpfdfcd"
status: in-progress
priority: high
type: feature
tags: ["cli", "reporting"]
created: "2026-04-18"
---

# Result comparison across runs

## Objective

Add a `skival compare` CLI command that loads two result directories and produces a diff report showing how treatments changed between runs. This closes the iteration feedback loop — users tweaking skills or prompts need to quickly see what improved, regressed, or stayed the same across correctness, cost, and duration.

## Tasks

- [x] Create `internal/compare/` package with a `Compare(baseline, candidate SuiteResult)` function that computes per-treatment deltas
- [x] Compute deltas for: pass rate (percentage points), median cost (USD and %), median duration (ms and %)
- [x] Handle mismatched evals/treatments gracefully (report added/removed treatments, skip unmatched)
- [x] Add markdown comparison report output with delta columns and directional indicators (↑/↓/=)
- [x] Add JSON comparison report output for programmatic consumption
- [x] Add `skival compare <baseline-dir> <candidate-dir>` Cobra command with `--format` flag
- [x] Add tests for comparison logic (matching treatments, mismatched treatments, identical results)
- [x] Add tests for the CLI command
- [x] Update CLI documentation (cli.md) with the new command

## Acceptance Criteria

- `skival compare results/run-1 results/run-2` outputs a readable diff showing per-eval and per-treatment changes
- Deltas show both absolute and percentage changes for cost and duration
- Pass rate changes shown as percentage point differences
- Added/removed evals or treatments are clearly labeled rather than causing errors
- Both `--format markdown` and `--format json` are supported
- Command returns non-zero exit code if baseline or candidate directory is missing/invalid
