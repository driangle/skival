---
id: "01m03xmkx"
title: "End-to-end deny test, docs, and examples for tool-access enforcement"
status: completed
priority: high
effort: medium
parent: "01m03wx7s"
phase: phase-1
dependencies: ["01m039wnp", "01m032mr5"]
tags: ["tool-access", "runner", "security", "docs"]
created_at: 2026-08-15
completed_at: 2026-08-21
---

# End-to-end deny test, docs, and examples for tool-access enforcement

> Sub-task of [[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]].
> Depends on [[01m039wnp-generate-per-variant-permission-config-and-wire-in]] and
> [[01m032mr5-default-deny-baseline-and-deprecate-advisory-disal]].

## Objective

Lock in the enforcement guarantee with an automated end-to-end check that an
unlisted built-in is actually denied, and bring the documentation and examples in
line with the new default-deny posture.

## Tasks

- [x] Add an end-to-end check that a variant with `allowed_tools: [Read, Grep]`
      cannot use an unlisted built-in (e.g. `Bash`) against a real agent invocation
- [x] Update `docs/configuration.md` tool-access section (~line 441) to describe
      generated deny-by-default enforcement and the unset-`allowed_tools` posture
- [x] Update `examples/runner-config` to reflect the new behaviour
- [x] Update `evals/prompts/tool-access.md` example

## Acceptance Criteria

- Behaviour is verified against a real agent run, not only unit tests
- `docs/configuration.md`, `examples/runner-config`, and
  `evals/prompts/tool-access.md` describe the generated deny-by-default enforcement
  and the documented default posture when `allowed_tools` is unset
- Unblocks [[01m037pwn]] (tool_not_used verifier as a leakage backstop)
