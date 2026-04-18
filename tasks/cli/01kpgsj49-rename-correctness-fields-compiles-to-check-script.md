---
title: "Rename correctness fields: compiles to check, script to check_output"
id: "01kpgsj49"
status: completed
priority: high
type: chore
tags: ["refactor", "correctness"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# Rename correctness fields: compiles to check, script to check_output

## Objective

Rename the two shell-command correctness verifiers to better reflect what they actually do:
- `compiles` → `check` — runs a shell command against the working directory (no stdin)
- `script` → `check_output` — runs a shell command with the agent's text output piped to stdin

The current names are misleading: `compiles` implies compilation but accepts any shell command; `script` is too generic and doesn't convey that the agent's output is piped to stdin.

## Tasks

- [x] Rename `Compiles` to `Check` in `Correctness` struct (`internal/suite/suite.go`) and update YAML tag to `check`
- [x] Rename `Script` to `CheckOutput` in `Correctness` struct and update YAML tag to `check_output`
- [x] Rename `CompilesVerifier` → `CheckVerifier` and rename file `internal/verifier/compiles.go` → `check.go`
- [x] Rename `ScriptVerifier` → `CheckOutputVerifier` and rename file `internal/verifier/script.go` → `check_output.go`
- [x] Update `BuildPipeline()` in `internal/verifier/pipeline.go` to use new field/type names
- [x] Update all `suite.yaml` examples under `examples/` that use `compiles` or `script`
- [x] Update all tests referencing old field names
- [x] Update validation logic if it references field names by string
- [x] Update report/output code if it displays verifier step names

## Acceptance Criteria

- [x] `correctness.check` runs a shell command in the eval dir, passes if exit 0 (same behavior as old `compiles`)
- [x] `correctness.check_output` runs a shell command with agent output on stdin, passes if exit 0 (same behavior as old `script`)
- [x] Old field names `compiles` and `script` no longer appear in source code or examples
- [x] All existing tests pass with the new names
- [x] `TestLoad_Examples` passes (all example suites still load correctly)
