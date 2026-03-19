---
title: "Markdown and JSON report generation"
id: "01km2hm3v"
status: pending
priority: medium
type: feature
tags: ["phase-3", "reporting"]
dependencies: ["01km2hm2y"]
created: "2026-03-19"
phase: phase-3
---

# Markdown and JSON report generation

## Objective

Generate structured reports from eval results in both Markdown (human-readable) and JSON (machine-readable) formats. Wire into the `skival run` and `skival report` commands via --format flag.

## Tasks

- [ ] Implement Markdown report in `internal/report/markdown.go`: results table (eval rows, treatment columns), ranking table, per-treatment detail sections
- [ ] Implement JSON report in `internal/report/json.go`: serialize SuiteResult with all metrics
- [ ] Wire --format flag: "markdown" (default) or "json"
- [ ] `skival report <results-dir>` reads persisted results and regenerates reports
- [ ] Markdown table formatting: align columns, format cost as $X.XXXX, duration as Xs

## Acceptance Criteria

- `skival run suite.yaml` prints a readable markdown summary to stdout by default
- `--format json` outputs valid JSON that can be piped to jq
- Results table shows pass/fail + cost + duration per treatment per eval
- Ranking table shows composite score breakdown
