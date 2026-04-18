---
title: "Configurable retry logic for flaky runs"
id: "01kpfdn43"
status: completed
priority: high
type: feature
tags: ["executor", "reliability"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# Configurable retry logic for flaky runs

## Objective

Add configurable retry logic so transient failures (timeouts, runner crashes, network errors) don't permanently mark a sample as failed. Each sample run can be retried with backoff, and users control whether only transient errors or all failures (including correctness) trigger retries.

### Configuration

```yaml
defaults:
  retry:
    max_attempts: 3          # total attempts including the first (default: 1, no retries)
    backoff: exponential     # fixed | exponential (default: fixed)
    delay: 2s                # base delay between retries (default: 2s)
    on: transient            # transient | all (default: transient)
```

- **`transient`** — retry on runner errors, timeouts, and network failures. Don't retry if the agent ran successfully but produced incorrect output.
- **`all`** — retry everything including correctness failures. Useful for flaky evals or giving the agent another chance.

Retry config can be set at suite defaults, per-eval, or per-treatment level following the existing override precedence.

## Tasks

- [x] Add `Retry` struct to suite schema (`MaxAttempts`, `Backoff`, `Delay`, `On`) with YAML parsing
- [x] Add validation rules: `max_attempts >= 1`, `backoff` enum, `delay` parseable as duration, `on` enum
- [x] Implement retry loop in executor around the sample run, respecting `max_attempts`
- [x] Implement fixed and exponential backoff with jitter
- [x] Classify run outcomes: transient error (runner crash, timeout, network) vs correctness failure vs success
- [x] When `on: transient`, only retry on transient errors; when `on: all`, retry on any non-pass outcome
- [x] On retry, use the best result (pass > fail, lower cost breaks ties) rather than the last attempt
- [x] Add retry metadata to `RunResult` (attempt number, total attempts, whether it was a retry)
- [x] Add tests for retry logic: backoff timing, transient vs all modes, result selection
- [x] Update suite loader tests for retry config parsing and validation
- [x] Update documentation (configuration.md, getting-started.md) with retry config

## Acceptance Criteria

- Setting `retry.max_attempts: 3` causes failed samples to be retried up to 2 additional times
- `on: transient` does not retry when the agent runs successfully but fails correctness checks
- `on: all` retries on any non-pass outcome including correctness failures
- Exponential backoff doubles the delay each attempt (with jitter)
- Fixed backoff waits the same `delay` between each attempt
- The best result across attempts is kept (not the last)
- Default behavior (no retry config) is unchanged — single attempt, no retries
- Retry config inherits via the standard precedence: treatment > eval > defaults
