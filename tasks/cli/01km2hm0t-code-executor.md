---
title: "Code executor for generated output"
id: "01km2hm0t"
status: pending
priority: medium
type: feature
tags: ["phase-2", "verifier"]
dependencies: ["01km2hkyh"]
created: "2026-03-19"
phase: phase-2
---

# Code executor for generated output

## Objective

Implement a code executor that runs generated code from the agent's output when correctness.execute is true. Captures stdout/stderr and exit code for downstream verification.

## Tasks

- [ ] Implement executor in `internal/executor/executor.go` — extracts code from RunOutput, writes to temp file, executes it
- [ ] Support configurable timeout from correctness.timeout (default 30s)
- [ ] Capture stdout, stderr, and exit code in an ExecutionResult struct
- [ ] Detect language/runtime from the eval config or code fencing (e.g., go run, node, python)
- [ ] Clean up temp files after execution

## Acceptance Criteria

- Executor writes generated code to a temp file and runs it with the appropriate runtime
- stdout and stderr are captured separately
- Execution respects the configured timeout and returns an error on timeout
- Temp files are cleaned up even on failure
