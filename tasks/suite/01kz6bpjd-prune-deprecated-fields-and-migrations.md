---
id: "01kz6bpjd"
title: "Prune deprecated fields and schedule migration-pass removal"
status: pending
priority: low
effort: medium
type: chore
tags: ["suite", "tech-debt", "yaml-api"]
created: 2026-08-04
---

# Prune deprecated fields and schedule migration-pass removal

## Objective

`Load` in `internal/suite/loader.go` runs five migration/deprecation passes —
`migrateStateToProbes`, `migrateCorrectnessToVerify`, `migrateAllowedTools`
(plus state→probes and the correctness sub-conversions) — and the deprecated
fields still live in the structs (`Correctness`, `Output`, `StateAssertion`,
`Variant.AllowedTools`). That is a lot of backward-compat baggage for a pre-1.0
tool depending on `agentrunner v0.0.1`, and it enlarges the schema surface that
docs and validation must cover.

## Tasks

- [ ] Inventory every deprecated field and its migration pass, with the version
      it was deprecated in
- [ ] Confirm all `examples/` and docs use only the current schema (depends on
      the docs-alignment task)
- [ ] Pick a removal version and record it (CHANGELOG / migration note)
- [ ] Remove the deprecated struct fields and their migration passes at that
      version, keeping a clear load-time error that points users to the new field
- [ ] Delete now-dead migration tests and add "removed field errors clearly" tests

## Acceptance Criteria

- A documented decision exists for when each deprecated field is removed
- After removal, deprecated fields produce an actionable error, not silent drop
- No `examples/` or docs depend on removed fields

## Notes

Depends on `01kz6nr0f` (docs must stop referencing deprecated fields first) and
should be coordinated with `01kz63dv0` (strict decoding changes how removed
fields surface).
