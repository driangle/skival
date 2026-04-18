---
title: "Parallel execution for evals and samples"
id: "01kpfdfbx"
status: in-progress
priority: high
type: feature
tags: ["performance", "executor"]
created: "2026-04-18"
---

# Parallel execution for evals and samples

## Objective

Enable parallel execution of independent work units (evals, treatments, samples) to reduce wall-clock time for large suites. Currently the executor runs everything sequentially: outer loop over evals, inner loop over treatments, innermost loop over samples. For suites with many evals or high sample counts, this makes runs unnecessarily slow since most work units are independent.

## Tasks

- [x] Add a `--parallel` / `-p` CLI flag to `skival run` controlling max concurrency (default: 1 for backwards compatibility)
- [x] Add an optional `parallel` field to suite-level defaults in the YAML schema
- [x] Refactor `executeEval` to run samples concurrently using a worker pool (bounded by concurrency limit)
- [x] Ensure per-sample directory isolation works correctly under concurrency (each goroutine gets its own temp dir)
- [x] Make progress reporting thread-safe (protect stderr writes with synchronization)
- [x] Make result collection thread-safe (aggregate `RunResult` slices safely across goroutines)
- [x] Ensure lifecycle hooks run in the correct order: `before` once → parallel samples (each with its own `reset`) → `after` once
- [x] Add concurrency-aware tests to the executor package
- [x] Update documentation (cli.md, configuration.md) with the new flag and field

## Acceptance Criteria

- Running `skival run suite.yaml --parallel 4` executes up to 4 samples concurrently
- Sequential mode (`--parallel 1` or omitted) produces identical results to current behavior
- Progress output remains coherent under concurrency (no interleaved lines)
- Per-sample isolation directories are independent and cleaned up correctly
- Hook execution order is preserved: `before` → N parallel samples → `after`
- All existing tests continue to pass
