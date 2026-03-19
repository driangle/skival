---
title: "Result types and basic execution flow"
id: "01km2hkxn"
status: completed
priority: high
type: feature
tags: ["phase-1", "core"]
dependencies: ["01km2hkvf", "01km2hkwf"]
created: "2026-03-19"
phase: phase-1
---

# Result types and basic execution flow

## Objective

Define result types (EvalResult, TreatmentResult) and implement the core execution flow that loads a suite, iterates evals and treatments, invokes the runner, and collects results. Wire this into the `skival run` command.

## Tasks

- [x] Define result types in `internal/result/result.go`: EvalResult, TreatmentResult, RunResult, SuiteResult
- [x] Implement execution orchestrator that iterates evals → treatments → samples and calls the runner
- [x] Print raw results to console (treatment name, pass/fail placeholder, cost, duration)
- [x] Wire the orchestrator into `apps/cli/cmd/run.go` so `skival run suite.yaml` loads and executes
- [x] Test with a minimal suite.yaml against a real Claude Code CLI invocation

## Acceptance Criteria

- `skival run suite.yaml` loads the suite, runs each treatment, and prints results to stdout
- Results include cost_usd and wall_clock_ms from the runner
- Execution runs control first, then variations in order
- Errors in individual runs don't crash the entire suite
