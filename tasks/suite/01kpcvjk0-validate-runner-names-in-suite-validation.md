---
title: "Validate runner names in suite validation"
id: "01kpcvjk0"
status: completed
priority: high
type: feature
effort: small
tags: ["backend", "suite"]
dependencies: ["01kpcvhw5"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Validate runner names in suite validation

## Objective

Add validation for runner names in `validate.go` so that unknown runner values are caught at load time rather than failing at execution.

## Tasks

- [x] Define a `validRunners` set containing `""`, `"claude-code"`, and `"ollama"`
- [x] Validate `defaults.runner`, each `eval.runner`, and each `treatment.runner` against the set
- [x] Produce a clear error message including the eval/treatment context prefix for unknown runners

## Acceptance Criteria

- [x] A suite with `runner: unknown` on a treatment produces a validation error
- [x] A suite with `runner: claude-code` or `runner: ollama` passes validation
- [x] A suite with no `runner` field passes validation (empty string is valid)
- [x] Error messages include the eval ID and treatment name for context
- [x] Unit tests cover valid runners, unknown runners, and empty runner
