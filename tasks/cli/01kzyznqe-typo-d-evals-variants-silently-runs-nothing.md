---
title: "Typo'd --evals/--variants silently runs nothing"
id: "01kzyznqe"
status: completed
priority: high
type: bug
tags: ["cli", "filtering", "ux"]
created: "2026-08-13"
phase: phase-4
completed_at: 2026-08-13
---

# Typo'd --evals/--variants silently runs nothing

## Steps to Reproduce

1. Run a suite with a misspelled filter id, e.g. `--evals add-bugs` (valid id is
   different) or `--variants plugn-skill` (typo of `plugin-skill`).
2. Observe the command prints an empty results table and exits 0.

## Expected Behavior

An `--evals` / `--variants` value that matches no known id should be a **hard
error** that lists the valid ids, e.g.:

```
error: no eval matches "add-bugs". Valid evals: add-dependency, remove-bug, ...
```

The command should exit non-zero so a typo can't be mistaken for a completed run.

## Actual Behavior

`--evals add-bugs` and `--variants plugn-skill` both print an empty results
table and exit 0. On a long-running suite you'd walk away and come back to
nothing, with no indication the filter never matched.

## Tasks

- [x] Validate `--evals` and `--variants` values against the suite's known ids before running.
- [x] On any unmatched id, fail with a non-zero exit and an error listing the valid ids.
- [x] Add tests: unmatched `--evals`/`--variants` values produce a hard error listing valid ids and a non-zero exit.

## Acceptance Criteria

- Passing an `--evals` or `--variants` id that matches nothing exits non-zero.
- The error message lists the valid ids for that flag.
- A partially-valid list (some ids match, some don't) also errors on the unmatched ids rather than silently dropping them.
- Tests cover the unmatched-id error path for both flags.

## Environment

- OS:
- Version:
