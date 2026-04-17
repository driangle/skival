---
title: "Per-sample working directory isolation"
id: "01kpcv05j"
status: completed
priority: low
type: feature
tags: ["schema", "execution"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Per-sample working directory isolation

## Objective

Add built-in support for isolating each sample's working directory so destructive evals (file deletions, DB mutations, git operations) don't leak state between samples. Currently `setup.reset` hooks handle this manually, but it's error-prone and requires users to write cleanup scripts for every eval.

A `dir_template` or `isolate: true` option should clone/copy the eval's working directory into a temporary directory per sample, ensuring each run starts from identical state.

## Tasks

- [x] Design the isolation mechanism (copy vs git worktree vs symlink + overlay)
- [x] Add `isolate: bool` field to `Eval` struct in `internal/suite/suite.go`
- [x] Implement per-sample directory creation in `internal/executor/executor.go` (copy eval dir to temp dir before each run)
- [x] Pass the isolated directory as the working dir to the runner
- [x] Clean up temporary directories after each sample completes
- [x] Ensure verifier scripts run in the correct (isolated) directory
- [x] Add suite loader validation and tests
- [x] Add executor tests for isolation behavior

## Acceptance Criteria

- When `isolate: true`, each sample runs in its own copy of the eval directory
- Files modified by sample 1 are not visible to sample 2
- Temporary directories are cleaned up after the eval completes
- `setup.reset` hooks still work and run in the isolated directory
- Existing suites without `isolate` behave unchanged
- Verification scripts receive the correct working directory
