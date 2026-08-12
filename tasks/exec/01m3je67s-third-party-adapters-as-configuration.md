---
id: "01m3je67s"
title: "Third-party agent adapters as configuration only"
status: pending
priority: medium
effort: medium
type: docs
tags: ["exec", "examples", "differentiation"]
created: 2026-08-12
dependencies: ["01m3kwx3g", "01m38j703"]
context: ["examples/", "docs/exec-runner.md"]
verify:
  - type: bash
    run: "make validate-examples"
  - type: assert
    check: "At least three third-party agent CLIs are evaluated end to end with no Go code in skival"
---

# Third-party agent adapters as configuration only

## Objective

The differentiation strategy claims skival can evaluate any agent without code
changes. Right now that claim rests on one example (`examples/exec-python`),
while README simultaneously advertises Codex and Aider support that does not
exist in the registry.

Shipping working adapters for real third-party agent CLIs — as `runner_config`
blocks and nothing else — is direct evidence for the positioning, and costs one
YAML file each instead of one Go package each. It also resolves the Codex and
Aider overclaim honestly: they become supported *through the contract*, not
through bespoke integrations.

## Tasks

- [ ] Add example suites driving at least three third-party agent CLIs via
      `exec` (candidates: codex CLI, aider, opencode, cursor-agent)
- [ ] Include a small shim per agent only where its output is not already JSONL,
      kept in the example directory rather than in skival
- [ ] Document each adapter's measurable capabilities (cost? usage? tool calls?)
      so users know which metrics are available
- [ ] Add a docs page: "evaluate any agent" walking from a bare shell command to
      a fully conformant adapter
- [ ] Keep all adapter suites in `make validate-examples`

## Acceptance Criteria

- Three or more third-party CLIs are evaluated with no Go code in skival
- Each adapter documents which metrics it can and cannot report
- No runner is advertised that is not either registered or demonstrated
