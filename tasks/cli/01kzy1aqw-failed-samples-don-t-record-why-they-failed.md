---
title: "Failed samples don't record why they failed"
id: "01kzy1aqw"
status: completed
priority: high
type: bug
tags: ["reporting", "verification"]
created: "2026-08-13"
phase: phase-3
completed_at: 2026-08-13
---

# Failed samples don't record why they failed

## Steps to Reproduce

1. Run a suite where some samples fail verification.
2. Open the persisted run JSON for a failed sample (e.g. `run-3.json`).
3. Observe it contains `"pass": false` and nothing else — no failing step name, no verifier stdout/stderr.

## Expected Behavior

A failed sample's persisted result should record *why* it failed. Per-step
results already exist at runtime (the live output prints `verify check: PASS`),
so they should be persisted rather than discarded:

- Persist per-step `{name, type, pass, exit_code, stdout, stderr}` on each run.
- Surface the first failing step's message in the report — either in the report
  table or a dedicated "Failures" section — e.g.
  `add-dependency: task file has no "## Objective" section`.

The goal: reading the report should tell the developer the failure reason
directly, instead of forcing them to find the temp workdir and re-run the
grader by hand.

## Actual Behavior

The persisted run for a failed sample records only `"pass": false`. The
per-step result computed at runtime is discarded, so there is no failing step
name, exit code, or verifier stdout/stderr anywhere in the report. Diagnosing
why 4/9 samples failed required locating the temp workdir referenced in the
report, `cd`-ing into it, and re-running the grader manually — the single
biggest time sink reported in this feedback.

## Tasks

- [x] Persist per-step results `{name, type, pass, exit_code, stdout, stderr}` on each run/sample record.
- [x] Include the first failing step's name and message in the report table or a "Failures" section.
- [x] Add tests covering a failed sample: assert per-step details are persisted and the failure reason is surfaced in the report.

## Acceptance Criteria

- A failed sample's persisted JSON includes per-step results with name, type, pass, exit code, and captured stdout/stderr.
- The generated report shows the first failing step's name and message for each failed sample.
- Tests verify that failure details are persisted and rendered.

## Environment

- OS:
- Version:
