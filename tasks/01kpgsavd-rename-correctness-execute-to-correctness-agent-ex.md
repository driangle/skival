---
title: "Rename correctness.execute to correctness.agent_exits_ok"
id: "01kpgsavd"
status: completed
priority: low
type: chore
tags: ["naming", "correctness"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# Rename correctness.execute to correctness.agent_exits_ok

## Objective

Rename the `correctness.execute` field to `correctness.agent_exits_ok` to better communicate what it checks — that the agent process exited with code 0, not that it runs something.

## Tasks

- [x] Rename `Execute` field to `AgentExitsOK` in `suite.Correctness` struct and update YAML tag
- [x] Update `BuildPipeline()` in `internal/verifier/pipeline.go` to reference new field
- [x] Update all `suite.yaml` files (examples, tests) to use `agent_exits_ok`
- [x] Update loader/validation logic if it references the field by name
- [x] Update documentation referencing `execute`

## Acceptance Criteria

- `correctness.agent_exits_ok: true` works identically to the old `correctness.execute: true`
- All existing tests pass
- `correctness.execute` is no longer recognized (no backwards compat shim)
