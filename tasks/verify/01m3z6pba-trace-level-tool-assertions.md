---
id: "01m3z6pba"
title: "Trace-level tool assertions (tool_used, tool_not_used, tool_sequence)"
status: pending
priority: high
effort: medium
type: feature
tags: ["verifier", "differentiation"]
created: 2026-08-12
context: ["internal/verifier/tool_activity.go", "internal/suite/verify.go", "docs/verifiers.md"]
verify:
  - type: bash
    run: "go test ./internal/verifier/... ./internal/suite/..."
  - type: assert
    check: "A run that never calls a given tool fails a tool_used step and passes a tool_not_used step"
---

# Trace-level tool assertions

## Objective

Every verifier except `judge` inspects the agent's final text or the filesystem.
There is no way to assert on what the agent actually *did*. promptfoo's
`skill-used` / `not-skill-used` assertions close exactly this gap, and it is the
one real capability difference in their favour.

skival already captures the full conversation (`RunResult.Conversation`) and
already summarizes tool calls (`internal/verifier/tool_activity.go`). The data
is there; only the verifier types are missing.

This matters beyond parity: "did the skill fire at all" is a different and often
more useful question than "was the answer good", and it is much cheaper and more
deterministic to check than an LLM judge.

## Tasks

- [ ] Add `tool_used` and `tool_not_used` verify types matching on tool name
- [ ] Add `tool_call_count` with a comparison operator and bound
- [ ] Add `tool_sequence` asserting an ordered subsequence of tool calls
- [ ] Extend `verifyTypeFields` so fields that do not belong to a step's type
      remain rejected at validation time
- [ ] Where a runner exposes first-class skill invocation events, normalize them
      so a single assertion works across runners
- [ ] Document the new types in `docs/verifiers.md` with a worked example

## Acceptance Criteria

- A run that never invokes a tool fails `tool_used` for it
- Assertions work identically for `claude-code` and for an `exec` runner
  emitting the JSONL event stream
- Validation rejects malformed steps
