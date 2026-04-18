---
title: "Improve error reporting on hook failures"
id: "01kpfdqw8"
status: completed
priority: medium
type: feature
tags: ["executor", "error-handling"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# Improve error reporting on hook failures

## Objective

When a setup hook (`before`, `reset`) fails mid-eval, remaining treatments for that eval are silently skipped. The current behavior of stopping the eval is correct — a failed setup means the environment is in an unknown state and continuing would produce unreliable results. However, the user gets no clear indication of what happened or which treatments were skipped.

Improve error messaging so hook failures are clearly reported in both stderr progress output and the final report, without changing the fail-fast behavior.

## Tasks

- [x] Capture hook stderr/stdout on failure and include it in the error message
- [x] Log a clear warning to stderr when treatments are skipped due to a hook failure (e.g., "Skipping 3 remaining treatments for eval 'foo': before hook failed")
- [x] Include skipped treatments in the eval result with a clear skip reason rather than omitting them entirely
- [x] Add an error/skip summary section to the markdown report listing evals with hook failures and which treatments were skipped
- [x] Add skip metadata to the JSON report for programmatic consumption
- [x] Add tests for error reporting on before/reset/after hook failures
- [x] Add tests that skipped treatments appear in the report with the correct reason

## Acceptance Criteria

- A `before` hook failure logs a clear message to stderr naming the eval and listing skipped treatments
- A `reset` hook failure logs which sample and treatment were affected
- Hook error output (stderr/stdout) is included in the error message, not swallowed
- Skipped treatments appear in the final report with a skip reason, not silently omitted
- The eval still stops on hook failure — no `continue_on_error` behavior
- The `after` hook still runs even if `before` or `reset` failed (cleanup opportunity)
