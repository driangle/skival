---
id: "01kpj1k9h"
title: "Remove complexity field from eval"
status: pending
priority: low
dependencies: []
tags: ["cleanup"]
created_at: 2026-04-19
---

# Remove complexity field from eval

## Objective

Remove the `complexity` field from `Eval`. It is validated and printed in `skival validate` output but has no runtime effect — nothing in the executor, verifier, or reporting reads it. Dead weight in the schema.

## Tasks

- [ ] Remove `Complexity` field from `Eval` struct in `internal/suite/suite.go`
- [ ] Remove `validComplexities` map and complexity validation in `internal/suite/validate.go`
- [ ] Remove complexity printing in `apps/cli/cmd/validate.go`
- [ ] Remove `complexity` from all example suite.yaml files
- [ ] Remove complexity-related tests in `internal/suite/validate_test.go`
- [ ] Ensure all tests pass

## Acceptance Criteria

- `complexity` key in YAML is silently ignored (no parse error)
- No references to `Complexity` remain in Go source
- All tests pass
