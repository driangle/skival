---
id: "01kz6mdp0"
title: "Reconcile isolate default (true) with docs and cost"
status: pending
priority: low
effort: small
type: docs
tags: ["docs", "suite", "executor"]
created: 2026-08-04
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

- [ ] Confirm the intended default for `isolate` (currently `true`)
- [ ] If keeping `true`: update README/docs to state isolation is on by default
      and how to disable it (`isolate: false`), and note the copy cost
- [ ] If switching to `false`: change the default in `mergeDefaults` and update
      any examples/tests that rely on implicit isolation
- [ ] Ensure isolated temp dirs are cleaned up after a sample (verify no leak of
      `skival-isolate-*` temp dirs)

## Acceptance Criteria

- Documented default for `isolate` matches `mergeDefaults`
- Users can find how to toggle isolation and understand its cost
- No leftover `skival-isolate-*` temp directories after a run
