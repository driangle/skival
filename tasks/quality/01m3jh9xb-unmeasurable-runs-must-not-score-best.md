---
id: "01m3jh9xb"
title: "Stop scoring unmeasurable runs as best on cost and duration"
status: pending
priority: high
effort: small
type: bug
tags: ["report", "ranking", "measurement"]
created: 2026-08-12
context: ["internal/report/rank.go"]
verify:
  - type: bash
    run: "go test ./internal/report/..."
  - type: assert
    check: "A variant whose runs all error does not receive a cost or duration score"
---

# Stop scoring unmeasurable runs as best on cost and duration

## Objective

`ratioLowerBetter` (`internal/report/rank.go:255`) returns `1.0` when a value is
`<= 0`. A variant that cannot execute reports $0.00 and 0ms, and therefore banks
full marks on both cost (28%) and duration (12%):

```
RANK  VARIANT         SCORE  PASS RATE  MEDIAN COST  MEDIAN DURATION
#1    works           1.000  100%       $0.0000      0ms
#2    totally-broken  0.400  0%         $0.0000      0ms
```

Missing data is being treated as optimal data. With three or more variants where
the working one costs real money, this can invert the ordering.

## Tasks

- [ ] Exclude runs with no usable cost/duration measurement from the per-eval
      normalization instead of scoring them 1.0
- [ ] Distinguish "genuinely free" (a local runner reporting $0) from
      "unmeasured" — the former is a real measurement, the latter is absent data
- [ ] Omit rather than fabricate the metric when a variant has no measurable run
- [ ] Show unmeasured metrics as `—` in reports, not as `$0.0000` / `0ms`

## Acceptance Criteria

- A variant whose runs all error does not outscore a working variant on any
  metric it never produced
- A local runner legitimately reporting zero cost still competes on cost
