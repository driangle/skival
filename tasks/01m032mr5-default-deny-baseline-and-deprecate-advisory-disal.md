---
id: "01m032mr5"
title: "Default-deny baseline and deprecate advisory disallowed_tools complement"
status: completed
priority: high
effort: medium
parent: "01m03wx7s"
phase: phase-1
dependencies: ["01m039wnp"]
tags: ["tool-access", "runner", "security"]
created_at: 2026-08-15
completed_at: 2026-08-17
---

# Default-deny baseline and deprecate advisory disallowed_tools complement

> Sub-task of [[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]].
> Depends on [[01m039wnp-generate-per-variant-permission-config-and-wire-in]].

## Objective

Make default-deny the baseline (folding in the old "hermetic mode" idea as the
implementation of default-deny rather than a separate toggle), and deprecate
reliance on the advisory `--disallowedTools` complement while keeping that flag path
working for runners that lack a permission-config mechanism.

## Design notes

- Define the posture when `allowed_tools` is unset: what a variant with no declared
  allow list may/may not do. Decide and document (baseline is deny-by-default per
  the parent task's framing — "hermeticity becomes the baseline, not a toggle").
- The `--disallowedTools` hardcoded complement has a staleness problem (new
  built-ins silently leak back in). Stop depending on it for enforcement; retain it
  only as a best-effort path for runners without a generated permission config.

## Tasks

- [x] Define and implement the default posture when `allowed_tools` is unset
      (default-deny baseline / hermetic-by-default)
- [x] Deprecate reliance on the advisory `--disallowedTools` complement for
      enforcement; keep the flag path working for runners without a permission-config
      mechanism
- [x] Tests covering the unset-`allowed_tools` default posture and the retained
      fallback flag path

## Acceptance Criteria

- With no `allowed_tools`, the documented default posture is applied and enforced,
  not silently permissive
- New built-ins do not leak in via a stale hardcoded complement
- Runners without a permission-config mechanism still receive the fallback flags
