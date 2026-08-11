---
title: "Add a dogfood self-test suite that evaluates the skival skill"
id: "01kzs0yfs"
status: completed
priority: high
type: feature
tags: ["dogfood", "skill", "eval", "ci"]
created: "2026-08-11"
completed_at: 2026-08-11
---

# Add a dogfood self-test suite that evaluates the skival skill

## Objective

Dogfood skival by using it to evaluate its own skill — as a **proper, first-class
part of the system**, not an example. Two layers, because dogfooding the skill
means running a real agent (the `claude-code` runner), which is non-deterministic,
costs money, and needs an API key:

1. **Deterministic layer (gates every PR, no LLM).** A Go test in the normal
   `go test ./...` path that guarantees the dogfood harness stays valid and the
   skill stays in sync with the validator.
2. **Agent-eval layer (opt-in, real LLM).** A separate manually/scheduled GitHub
   workflow that actually runs the baseline-vs-skill suite to measure whether the
   skill improves the agent's output. Never gates merges.

Correctness in the agent eval is measured objectively by running `skival validate`
on the agent's produced `suite.yaml` — the exact self-validation step the skill
instructs the agent to perform. The comparison is baseline (no skill) vs.
with-skill (real `SKILL.md` injected) on prompts that describe *what to evaluate*
but deliberately omit the schema, so the skill is the only source of schema
knowledge and the measurement is meaningful.

## Placement

- **Not** under `examples/`. Canonical dogfood assets live in a dedicated
  top-level `dogfood/` directory: `dogfood/suite.yaml`, `dogfood/workdir/`,
  `dogfood/verify.sh`, and prompt files.
- The deterministic Go test lives where `go test ./...` picks it up (e.g.
  `internal/dogfood/` or an integration test package), loading the canonical
  suite through the same loader/validator the CLI uses — no external binary,
  no shelling out.

## Tasks

### Deterministic layer (CI, every PR)
- [x] Write `dogfood/suite.yaml`: `baseline` (no skill) vs `with-skill`
      referencing the real `claude-code-plugin/skills/skival/SKILL.md`, with the
      `check`/`verify.sh` correctness step and `samples: 3`.
- [x] Add a Go test that (a) asserts `SKILL.md` exists and is a valid single skill
      (frontmatter `name`/`description`), (b) loads & validates `dogfood/suite.yaml`
      via the internal suite loader (asserting it passes), and (c) guards
      skill/validator drift (e.g. the verifier types & schema fields the skill
      documents still parse). This runs in the existing `test` CI job — no new
      always-on secrets or LLM calls.
- [x] Confirm the test runs green under `make check` / `go test ./...`.

### Agent-eval layer (opt-in workflow)
- [x] Author `dogfood/verify.sh` that runs `skival validate` on the
      agent-produced suite file; wire it as a `check` verifier alongside
      `agent_exits_ok`. Use `setup.reset` to clean the produced file between
      variants.
- [x] Write 2–3 varied task prompts (e.g. model comparison, skill A/B,
      tool-access comparison); each describes the task but not the schema.
- [x] Add `.github/workflows/dogfood.yaml` triggered by `workflow_dispatch`
      (and optionally a nightly `schedule`) that builds the binary, runs
      `skival run dogfood/suite.yaml --results-dir ...`, and uploads the results
      as an artifact. Reads `ANTHROPIC_API_KEY` from repo secrets. Does **not**
      run on `push`/`pull_request` and does not gate merges.
- [x] Document the dogfood harness (both layers + how to trigger the eval) in a
      `dogfood/README.md` and link it from the main README.

## Acceptance Criteria

- `dogfood/suite.yaml` exists, passes `skival validate`, and has a `baseline`
  (no skill) variant plus a `with-skill` variant injecting the real
  `claude-code-plugin/skills/skival/SKILL.md`.
- Nothing dogfood-related lives under `examples/`.
- A Go test validates the skill file and the canonical suite via the internal
  loader and runs as part of `go test ./...` (thus every PR) with no LLM calls
  and no required secrets.
- The agent eval's correctness check runs `skival validate` against the agent's
  generated suite file (via `check`/`verify.sh`); prompts omit the schema.
- `.github/workflows/dogfood.yaml` exists, is `workflow_dispatch` (± scheduled),
  uses an `ANTHROPIC_API_KEY` secret, uploads results, and does not trigger on
  push/PR.
- `make check` passes; the dogfood harness is documented.
