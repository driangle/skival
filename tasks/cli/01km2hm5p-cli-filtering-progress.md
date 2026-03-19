---
title: "CLI filtering and progress display"
id: "01km2hm5p"
status: pending
priority: low
type: feature
tags: ["phase-4", "ux"]
dependencies: ["01km2hkxn"]
created: "2026-03-19"
phase: phase-4
---

# CLI filtering and progress display

## Objective

Add --treatments and --evals flags to filter which treatments/evals are run, and display live progress during execution (eval N/M, treatment X, sample Y/Z).

## Tasks

- [ ] Implement --evals flag: comma-separated list of eval IDs to include
- [ ] Implement --treatments flag: comma-separated list of treatment names to include
- [ ] Display progress: current eval, treatment, sample number, elapsed time
- [ ] Show running cost total as runs complete
- [ ] Use terminal capabilities (carriage return) for in-place progress updates

## Acceptance Criteria

- `--evals foo,bar` only runs those two evals
- `--treatments control,with-skill` only runs those treatments
- Progress output shows which eval/treatment/sample is currently running
- Progress doesn't interfere with report output (use stderr for progress, stdout for report)
