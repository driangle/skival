---
title: "Add runner field to results and update reports"
id: "01kpcvjrw"
status: completed
priority: high
type: feature
effort: small
tags: ["backend", "report"]
dependencies: ["01kpcvjj5"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Add runner field to results and update reports

## Objective

Add a `Runner` field to `TreatmentResult` so reports can show which runner was used. Update the markdown and JSON reports to display runner names when a suite uses multiple runners.

## Tasks

- [x] Add `Runner string` field (with json tag) to `TreatmentResult` in `internal/result/result.go`
- [x] Set `Runner` on `TreatmentResult` during execution (in executor's `executeTreatment`)
- [x] Update `WriteMarkdown` to append `(runner-name)` to treatment headers when multiple runners are present in the suite
- [x] Update the rankings table to include a `Runner` column
- [x] Update `WriteJSON` to include the `runner` field in JSON output
- [x] Update existing report tests

## Acceptance Criteria

- [x] `TreatmentResult.Runner` is populated with the runner name used for that treatment
- [x] Markdown report shows runner name next to treatment name when the suite has more than one distinct runner
- [x] Markdown report omits runner annotation when all treatments use the same runner (cleaner output)
- [x] Rankings table includes a Runner column
- [x] JSON report includes `"runner"` field in treatment results
- [x] All existing report tests pass
