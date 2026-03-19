---
title: "Results directory persistence"
id: "01km2hm4s"
status: completed
priority: medium
type: feature
tags: ["phase-3", "reporting"]
dependencies: ["01km2hkxn"]
created: "2026-03-19"
phase: phase-3
---

# Results directory persistence

## Objective

When --results-dir is specified, persist all run data to a timestamped directory structure: per-run JSON, conversation history JSONL, aggregate results, and summary reports.

## Tasks

- [ ] Create timestamped output directory under --results-dir
- [ ] Write per-run results as `run-N.json` per treatment per eval
- [ ] Write conversation history as `run-N.conversation.jsonl` (from RunOutput.History)
- [ ] Write aggregate.json per treatment with aggregated metrics
- [ ] Write summary.md and summary.json at the top level
- [ ] Ensure directory structure matches PLAN.md spec

## Acceptance Criteria

- `--results-dir ./results` creates `results/<timestamp>/` with full directory tree
- Each run's metrics and conversation history are persisted as separate files
- `skival report results/<timestamp>/` can load these files and regenerate reports
- Files are written atomically (write to temp, rename) to avoid partial writes on crash
