---
title: "Report ordering buries rankings behind workdir/session paths"
id: "01kzy0w89"
status: completed
priority: medium
type: chore
tags: ["reporting", "ux"]
created: "2026-08-13"
phase: phase-3
completed_at: 2026-08-13
---

# Report ordering buries rankings behind workdir/session paths

## Description

The report ordering buries the payload. Rankings — the thing most readers
actually want — sit at the bottom, behind 27 Workdir lines and 27 Session
lines (one pair per sample). The reader has to scroll past all the path noise
to reach the result that matters.

Two changes:

- Move **Rankings** to immediately after **Results**, near the top.
- Collapse the per-sample Workdir/Session path lists into a single reference to
  the results dir (paths remain discoverable there, rather than being printed
  inline for every sample).

## Tasks

- [x] Reorder the report so Rankings appears immediately after Results.
- [x] Collapse the per-sample Workdir and Session path lists into a single pointer to the results dir instead of one line per sample.
- [x] Update any report tests/snapshots to reflect the new ordering and collapsed paths.

## Acceptance Criteria

- Rankings renders immediately after Results in the report.
- Per-sample Workdir/Session paths are no longer printed inline for every sample; the report points to the results dir instead.
- Report tests/snapshots are updated and passing.
