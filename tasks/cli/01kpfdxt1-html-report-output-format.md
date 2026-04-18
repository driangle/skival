---
title: "HTML report output format"
id: "01kpfdxt1"
status: completed
priority: low
type: feature
tags: ["reporting", "cli"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# HTML report output format

## Objective

Add an HTML report output format (`--format html`) for sharing eval results. Markdown and JSON cover CLI and programmatic use, but HTML with sortable tables and styling is better for sharing results with teammates or embedding in documentation.

## Tasks

- [x] Add `WriteHTML()` function in `internal/report/` that generates a self-contained HTML file (inline CSS, no external dependencies)
- [x] Include sortable tables for per-eval results and aggregate summary
- [x] Add directional color coding for metrics (green = good, red = bad)
- [x] Wire `html` as a valid `--format` option in `run` and `report` commands
- [x] Add tests for HTML report generation
- [x] Update CLI documentation (cli.md) with the new format option

## Acceptance Criteria

- `skival run suite.yaml --format html` outputs a valid, self-contained HTML document
- `skival report results/ --format html` generates HTML from saved results
- Tables are sortable by clicking column headers (minimal inline JS)
- The HTML file renders correctly when opened directly in a browser (no server needed)
- All data present in the markdown report is also present in the HTML report
