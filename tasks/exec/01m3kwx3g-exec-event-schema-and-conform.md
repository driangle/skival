---
id: "01m3kwx3g"
title: "Version the exec event schema and add `skival conform`"
status: pending
priority: medium
effort: medium
type: feature
tags: ["exec", "differentiation"]
created: 2026-08-12
dependencies: ["01m3x4yen"]
context: ["internal/runners/exec/events.go", "docs/exec-runner.md", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "go test ./internal/runners/exec/... ./apps/cli/..."
  - type: assert
    check: "skival conform reports actionable errors for a program that emits malformed or incomplete events"
---

# Version the exec event schema and add `skival conform`

## Objective

The `exec` runner is skival's strongest structural advantage: promptfoo
integrates SDK by SDK, while skival's contract is "any program, prompt via
stdin/env/argfile, optional JSONL event stream". Any agent can be evaluated with
zero code in skival.

Today that contract is described in prose in `docs/exec-runner.md` and enforced
only by whatever `events.go` happens to parse. A contract you can *validate* is
a much stronger claim than a list of supported SDKs — it turns
harness-agnosticism from marketing into something a user can check in ten
seconds.

## Tasks

- [ ] Define a versioned event schema (`schema_version`) covering message,
      tool-call, and terminal events, including cost, usage, and duration
- [ ] Publish it as a spec page with a JSON Schema artifact
- [ ] Add `skival conform <command>` that runs a program against a fixture
      prompt and reports which parts of the contract it satisfies
- [ ] Report degraded capability explicitly: "no cost reported → cost metrics
      unavailable for this runner" rather than defaulting to zero
- [ ] Version-check events at parse time with a clear error on mismatch

## Acceptance Criteria

- `skival conform` distinguishes a fully conformant agent from one that emits
  only stdout
- A program emitting no cost events is reported as cost-unmeasurable, not free
- The schema is versioned and documented
