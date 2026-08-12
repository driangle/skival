---
id: "01m3jbzwq"
title: "--fail-on regression CI gate"
status: pending
priority: medium
effort: small
type: feature
tags: ["cli", "ci", "differentiation"]
created: 2026-08-12
dependencies: ["01m36vrpc", "01m3f2ybx"]
context: ["apps/cli/cmd/run.go", "docs/cli.md"]
verify:
  - type: bash
    run: "go test ./apps/cli/..."
  - type: assert
    check: "A variant with a worse verdict on a gated metric exits non-zero; an inconclusive verdict does not"
---

# --fail-on regression CI gate

## Objective

The verdict from `01m36vrpc` is only useful in CI if it can fail a build. This
is where the single static binary pays off: one step, no runtime, no Docker.

The gate must fail on `worse` and pass on `inconclusive` — failing on
inconclusive would make every noisy run block a PR, which is exactly the
behaviour that trains people to ignore the tool.

## Tasks

- [ ] Extend `--fail-on` with a `regression` mode gated on verdicts
- [ ] Allow selecting which metrics gate (e.g. correctness only, or correctness
      and cost)
- [ ] Emit a short, copy-pasteable summary suitable for a PR comment
- [ ] Document the CI recipe in `docs/cli.md`

## Acceptance Criteria

- A real regression fails the build
- An inconclusive verdict does not fail the build
- The failure message names the metric, the delta, and the sample count
