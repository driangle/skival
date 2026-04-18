---
title: "Refactor expected_output to structured output object"
id: "01kpgt92g"
status: completed
priority: medium
type: feature
tags: ["correctness", "yaml-api"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# Refactor expected_output to structured output object

## Objective

Replace the flat `expected_output: [strings]` correctness field with a structured `output:` object. The current format is ambiguous — a list of strings doesn't communicate that it means "output must contain all of these substrings." The new structure makes semantics explicit and is extensible for future match modes.

**Before:**
```yaml
correctness:
  expected_output:
    - "PASS"
    - "all tests green"
```

**After:**
```yaml
correctness:
  output:
    contains:
      - "PASS"
      - "all tests green"
```

## Tasks

- [x] Add `Output` struct with `Contains []string` field to `suite.Correctness` in `internal/suite/suite.go`
- [x] Remove `ExpectedOutput` field from `Correctness` struct
- [x] Update YAML tag from `expected_output` to `output`
- [x] Update `mergeDefaults()` in loader to handle new nested structure
- [x] Update validation to check `output.contains` instead of `expected_output`
- [x] Update `BuildPipeline()` in `internal/verifier/pipeline.go` to read from `c.Output.Contains`
- [x] Update `OutputVerifier` in `internal/verifier/output.go` (no logic change, just wiring)
- [x] Update all example `suite.yaml` files that use `expected_output`
- [x] Update tests (loader tests, verifier tests, any integration tests)
- [x] Run `taskmd validate` and full test suite

## Acceptance Criteria

- `expected_output` field is no longer recognized in suite.yaml (validation rejects it)
- `output.contains` works identically to the old `expected_output` behavior (AND semantics, substring match)
- All examples load and pass `TestLoad_Examples`
- Verifier tests pass with the new structure
