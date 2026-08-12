---
id: "01m3ytgap"
title: "Judge the workspace diff, not just the final message"
status: pending
priority: high
effort: medium
type: feature
tags: ["verifier", "judge", "differentiation"]
created: 2026-08-12
dependencies: ["01m3bqx4q"]
context: ["internal/verifier/judge.go", "internal/executor/isolate.go", "docs/verifiers.md"]
verify:
  - type: bash
    run: "go test ./internal/verifier/..."
  - type: assert
    check: "The judge prompt for an isolated run includes the diff of files the agent changed"
---

# Judge the workspace diff, not just the final message

## Objective

`judgePromptTemplate` shows the judge the eval prompt, a tool-activity summary,
and the agent's final response. It never shows the code. For a tool that calls
itself an "AI coding skill evaluator", quality is being scored on the agent's
*summary of its work* rather than on the work.

The same limitation applies to comparative judging, which compares the final
text of one arbitrary passing sample.

Isolation already produces a clean copy of the workspace per sample, which gives
a before/after pair to diff. This is a genuine quality edge over every
competitor in the space, and it uses machinery that already exists.

## Tasks

- [ ] Capture the workspace diff produced by a run (snapshot before, diff after)
- [ ] Include the diff in the judge prompt, bounded by a configurable character
      cap like comparative judging already does
- [ ] Include the diff in comparative judging so quality is compared on the work
- [ ] Handle the non-isolated case explicitly rather than silently degrading —
      say the diff is unavailable
- [ ] Add a `diff_contains` verifier for the deterministic cases that do not
      need an LLM
- [ ] Persist the diff alongside the conversation for inspection

## Acceptance Criteria

- An isolated run's judge prompt contains the changed files
- Comparative judging compares diffs, not only final messages
- A non-isolated run reports that no diff was available rather than judging on
  the summary without saying so
