---
title: "Matrix syntax for cross-cutting treatment comparisons"
id: "01kpcv07b"
status: completed
priority: high
type: feature
tags: ["schema", "execution", "reporting"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Matrix syntax for cross-cutting treatment comparisons

## Objective

Add a `matrix` syntax to suite.yaml that generates treatments from the cartesian product of multiple dimensions (runner, model, skills, MCP servers, etc.). Currently comparing across multiple axes requires manually enumerating every combination as a separate treatment, which is verbose and error-prone.

Example desired syntax:
```yaml
matrix:
  runner: [claude-code, codex]
  model: [opus, sonnet]
  skills:
    - name: baseline
      files: []
    - name: with-tools
      files: [./skills/tool-use.md]
```
This would generate 2×2×2 = 8 treatments automatically.

## Tasks

- [x] Design the matrix schema — which fields can be varied, how combinations are named
- [x] Add `Matrix` type to `internal/suite/suite.go` with dimension definitions
- [x] Implement matrix expansion in suite loader: cartesian product → flat treatment list
- [x] Handle naming: auto-generate treatment names from dimension values (e.g., `claude-code_opus_baseline`)
- [x] Add validation: matrix and explicit treatments are mutually exclusive (or define merge semantics)
- [x] Tag each generated treatment with its dimension values for sliced reporting
- [x] Update the executor to pass dimension metadata through to results
- [x] Update reporting to support grouping/filtering by dimension (dimension metadata on treatments; full sliced reporting tracked in 01kpcv08d)
- [x] Add suite loader tests for matrix expansion
- [x] Add an example suite demonstrating matrix comparisons

## Acceptance Criteria

- [x] `matrix` field in suite.yaml generates treatments from the cartesian product of all dimensions
- [x] Generated treatment names are deterministic and human-readable
- [x] Each generated treatment carries metadata about which dimension values it uses
- [x] Validation rejects suites that define both `matrix` and `treatments` on the same eval
- [x] Reports can be sliced by any matrix dimension (dimension metadata preserved; full sliced views in 01kpcv08d)
- [x] Single-value dimensions work (no unnecessary duplication)
