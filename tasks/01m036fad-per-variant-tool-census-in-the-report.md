---
title: "Per-variant tool census in the report"
id: "01m036fad"
status: pending
priority: high
type: feature
tags: ["tool-access", "observability", "report"]
created: "2026-08-15"
effort: medium
---

# Per-variant tool census in the report

## Objective

Show **which tools each variant actually used**, per variant, in the report — a
count of every tool invoked, not just external/skill usage.

`TaskCreate ×10` sitting next to a variant named `no-skill` is instantly diagnostic.
Both leaks in the motivating run would have been obvious at a glance instead of
hand-grepping JSONL after the fact. This is the reporting side of observability: the
benchmark should make what the agent had access to (and used) visible in its own
output, so results can be trusted rather than taken on faith. Pairs with the
pre-flight warning [[01m030n6n-pre-flight-tool-leak-detection-from-system-init-ev]]
(what was *available*) — this shows what was *used*.

## Design notes

- The raw data is already present: every `tool_use` block is in
  `RunResult.Conversation` (`internal/result/result.go:25`). Today
  `internal/verifier/tool_activity.go` (`SummarizeToolActivity`, line 41) walks it
  and renders a text summary for the judge, handling both the nested claude-code
  shape (`writeNestedBlocks`, line 59) and the flat exec-runner shape
  (`writeFlatEvent`, line 72) — but it does **not count or aggregate**. A counter
  can follow the same two-shape traversal.
- Nothing records per-tool counts today: `RunResult` (`internal/result/result.go:11-34`)
  has no tool field, and the report's per-variant metrics
  (`internal/report/rankaccumulate.go:57-65`: PassRate, MedianCost, MedianDuration,
  MedianTokens, QualityScore) have no tool column.
- Plan:
  - Record a per-sample tool-count map (tool name → invocation count) on
    `RunResult`, populated once from the conversation (reuse/extend the
    `tool_activity.go` traversal so there's a single source of truth).
  - Aggregate to per-variant totals and surface in the markdown/JSON/HTML report
    (`internal/report/markdown.go`, `json.go`, `html*.go`).
- **Consider reusing vibeview** to inspect the session and produce the tool census
  rather than reimplementing extraction — skival or the client can run vibeview over
  the session JSONL. Evaluate before building bespoke aggregation.
- Don't assume a tool taxonomy; count whatever tool names appear. Keep the display
  compact (e.g. `Read ×12, Grep ×4, TaskCreate ×10`), sorted by count.

## Tasks

- [ ] Add a per-sample tool-count field to `RunResult` populated from the
      conversation (single traversal shared with `tool_activity.go`)
- [ ] Aggregate per-variant tool counts in the report accumulation
      (`internal/report/rankaccumulate.go`)
- [ ] Render the per-variant tool census in markdown, JSON, and HTML reports
- [ ] Evaluate reusing vibeview for extraction; record the decision
- [ ] Tests: counting from both conversation shapes (nested claude-code + flat exec),
      aggregation across samples, and report rendering

## Acceptance Criteria

- The report shows, for each variant, every tool it used with an invocation count
- Counts are correct for both the nested claude-code and flat exec-runner
  conversation shapes
- A variant that used a tool outside its intended set is visually obvious in the
  report without inspecting raw JSONL
- Works across markdown, JSON, and HTML report formats
