---
title: "Workdirs accumulate in /tmp unbounded; add --keep-workdirs"
id: "01kzyjtey"
status: completed
priority: medium
type: feature
tags: ["workdir", "cleanup", "ux"]
created: "2026-08-13"
phase: phase-1
completed_at: 2026-08-13
---

# Workdirs accumulate in /tmp unbounded; add --keep-workdirs

## Objective

Per-sample workdirs accumulate in `/tmp` unbounded — 27 copies for a single
run here — and are never cleaned up. The right ergonomic is a
`--keep-workdirs=failed` **default**: keep the workdirs you need to debug a
failure, drop the rest automatically.

## Tasks

- [x] Add a `--keep-workdirs` option with values `all` / `failed` / `none` (default `failed`).
- [x] After each sample (and/or at run end), remove workdirs that don't match the keep policy.
- [x] Ensure kept workdirs remain the ones referenced by the report / failure details so debugging still works (coordinate with failure-reporting work).
- [x] Add tests: `failed` keeps only failing samples' workdirs, `all` keeps everything, `none` cleans all.

## Acceptance Criteria

- `--keep-workdirs` accepts `all`, `failed`, and `none`, defaulting to `failed`.
- With the default, passing samples' workdirs are removed and failing samples' workdirs are retained.
- Retained workdirs are still the ones referenced by the report for debugging.
- Tests cover each keep policy.
