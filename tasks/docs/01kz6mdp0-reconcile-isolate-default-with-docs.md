---
id: "01kz6mdp0"
title: "Reconcile isolate default (true) with docs and cost"
status: completed
priority: low
effort: small
type: docs
tags: ["docs", "suite", "executor"]
created: 2026-08-04
completed_at: 2026-08-10
---

# Reconcile isolate default (true) with docs and cost

## Objective

`mergeDefaults` in `internal/suite/loader.go` defaults `Isolate` to `true` when
unset, so every eval copies its working directory per sample by default
(`createIsolatedDir` -> `copyDir` in `internal/executor/executor.go`). The README
describes this as opt-in — "**Working directory isolation** — *Optionally* copy
the eval directory per sample" — which understates the default behavior and its
per-sample disk/time cost.

Decide the intended default and make docs and behavior agree.

## Tasks

- [x] Confirm the intended default for `isolate` — decided: `false` (opt-in)
- [] ~~If keeping `true`: update docs to state on-by-default~~ (not chosen)
- [x] Switch the default to `false` in `mergeDefaults`; updated the loader test
      that asserted the implicit default. Also guarded `isolate: true` with no
      `dir` so it is a no-op instead of crashing (`copyDir("")`).
- [x] Isolated temp dirs: decided to **keep them preserved for inspection**
      (they surface in the report's Workdirs section); documented that they are
      intentionally left under `$TMPDIR/skival-isolate-*` and how to clean them.

## Acceptance Criteria

- Documented default for `isolate` matches `mergeDefaults`
- Users can find how to toggle isolation and understand its cost
- No leftover `skival-isolate-*` temp directories after a run
