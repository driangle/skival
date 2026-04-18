---
title: "Rename correctness fields: compiles to check, script to check_output"
id: "01kpgsj49"
status: pending
priority: high
type: chore
tags: ["refactor", "correctness"]
created: "2026-04-18"
---

# Rename correctness fields: compiles to check, script to check_output

## Objective

Rename the two shell-command correctness verifiers to better reflect what they actually do:
- `compiles` → `check` — runs a shell command against the working directory (no stdin)
- `script` → `check_output` — runs a shell command with the agent's text output piped to stdin

The current names are misleading: `compiles` implies compilation but accepts any shell command; `script` is too generic and doesn't convey that the agent's output is piped to stdin.

## Tasks

- [ ] Rename `Compiles` to `Check` in `Correctness` struct (`internal/suite/suite.go`) and update YAML tag to `check`
- [ ] Rename `Script` to `CheckOutput` in `Correctness` struct and update YAML tag to `check_output`
- [ ] Rename `CompilesVerifier` → `CheckVerifier` and rename file `internal/verifier/compiles.go` → `check.go`
- [ ] Rename `ScriptVerifier` → `CheckOutputVerifier` and rename file `internal/verifier/script.go` → `check_output.go`
- [ ] Update `BuildPipeline()` in `internal/verifier/pipeline.go` to use new field/type names
- [ ] Update all `suite.yaml` examples under `examples/` that use `compiles` or `script`
- [ ] Update all tests referencing old field names
- [ ] Update validation logic if it references field names by string
- [ ] Update report/output code if it displays verifier step names

## Acceptance Criteria

- [ ] `correctness.check` runs a shell command in the eval dir, passes if exit 0 (same behavior as old `compiles`)
- [ ] `correctness.check_output` runs a shell command with agent output on stdin, passes if exit 0 (same behavior as old `script`)
- [ ] Old field names `compiles` and `script` no longer appear in source code or examples
- [ ] All existing tests pass with the new names
- [ ] `TestLoad_Examples` passes (all example suites still load correctly)
