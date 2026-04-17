---
title: "Matrix syntax for cross-cutting treatment comparisons"
id: "01kpcv07b"
status: pending
priority: high
type: feature
tags: ["schema", "execution", "reporting"]
created: "2026-04-17"
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

- [ ] Design the matrix schema — which fields can be varied, how combinations are named
- [ ] Add `Matrix` type to `internal/suite/suite.go` with dimension definitions
- [ ] Implement matrix expansion in suite loader: cartesian product → flat treatment list
- [ ] Handle naming: auto-generate treatment names from dimension values (e.g., `claude-code_opus_baseline`)
- [ ] Add validation: matrix and explicit treatments are mutually exclusive (or define merge semantics)
- [ ] Tag each generated treatment with its dimension values for sliced reporting
- [ ] Update the executor to pass dimension metadata through to results
- [ ] Update reporting to support grouping/filtering by dimension
- [ ] Add suite loader tests for matrix expansion
- [ ] Add an example suite demonstrating matrix comparisons

## Acceptance Criteria

- `matrix` field in suite.yaml generates treatments from the cartesian product of all dimensions
- Generated treatment names are deterministic and human-readable
- Each generated treatment carries metadata about which dimension values it uses
- Validation rejects suites that define both `matrix` and `treatments` on the same eval
- Reports can be sliced by any matrix dimension
- Single-value dimensions work (no unnecessary duplication)
