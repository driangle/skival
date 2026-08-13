---
title: "Surface token usage in reports (persist + render)"
id: "01kzs8hp2"
status: completed
priority: medium
type: bug
tags: ["reporting", "persistence", "tokens"]
created: "2026-08-11"
completed_at: 2026-08-13
---

# Surface token usage in reports (persist + render)

## Summary

The README advertises that skival "Measures … **token usage**" (`README.md:6`),
but token counts never appear in any report. Usage is captured at runtime and
lives in memory during a run, but it is dropped before it can reach the user:
it is neither persisted to disk nor rendered by any report format.

## Steps to Reproduce

1. Run a suite that saves results, e.g.
   `skival run evals/suite.yaml --evals model-comparison --samples 1 --results-dir results/dogfood`.
2. Regenerate a report: `skival report results/dogfood/<run> --format html > report.html`.
3. Observe the report (and `summary.json`) show cost and duration but no token counts.

## Expected Behavior

Reports (markdown, JSON, and HTML) show token usage per variant — at minimum
median input and output tokens, ideally with cache-read / cache-creation
breakdown — alongside the existing cost column. Persisted results retain token
usage so `skival report` from disk shows the same numbers as a live run.

## Actual Behavior

No token counts anywhere in the report or in `summary.json`. Cost and duration
are the only per-run economic metrics surfaced.

## Root Cause

Usage plumbing exists end-to-end **up to** persistence/rendering, then stops:

- `agentrunner.Result.Usage` provides input/output/cache tokens. ✅
- `internal/executor/singlerun.go:66` populates `RunResult.Usage = res.Usage`, so
  tokens are in memory during a run. ✅
- `internal/persist/persist.go` (`runResultJSON`, ~lines 101–112) has `cost_usd`
  and `duration_ms` but **no usage/token fields** → tokens are dropped when
  writing `summary.json`, and `internal/persist/load.go` never reads them back. ❌
- `internal/report/*` builders and templates contain **no** reference to
  `token`/`usage` → no report format renders tokens even from an in-memory run. ❌

## Tasks

- [x] Persist usage: add token fields (input, output, cache_creation, cache_read,
      or an embedded `usage` object) to `runResultJSON` in `internal/persist/persist.go`
      (save) and `internal/persist/load.go` (load), round-tripping `agentrunner.Usage`.
- [x] Aggregate usage: extend `internal/result/aggregate.go` with median (and
      min/max where it fits the existing pattern) token metrics per variant.
- [x] Render usage in reports: add token columns/fields to the markdown, JSON,
      and HTML report builders/templates in `internal/report/` next to cost.
- [x] Add tests: persist round-trip retains usage; aggregate medians are correct;
      each report format includes the token figures.
- [x] Confirm cost and tokens stay consistent (cost is derived from the same
      usage) and update README/docs if the reported fields are named.

## Acceptance Criteria

- `summary.json` includes per-run token usage, and `skival report <dir>` shows the
  same token numbers as the originating `skival run`.
- Markdown, JSON, and HTML reports all display token usage per variant alongside
  cost.
- New tests cover the persist round-trip, the aggregate computation, and token
  presence in each report format; `make check` passes.
