---
title: "LLM judge verifier"
id: "01km2hm6j"
status: pending
priority: low
type: feature
tags: ["phase-4", "verifier"]
dependencies: ["01km2hm1r"]
created: "2026-03-19"
phase: phase-4
---

# LLM judge verifier

## Objective

Implement an LLM-as-judge verifier that uses a fast/cheap model (e.g., Claude Haiku) to evaluate whether the agent's output satisfies subjective correctness criteria that can't be checked mechanically.

## Tasks

- [ ] Implement JudgeVerifier in `internal/verifier/judge.go`
- [ ] Construct a judge prompt: include the eval's prompt, the agent's full output/conversation, and the correctness criteria
- [ ] Call a fast model (Haiku) via agentrunner to get a pass/fail + reason
- [ ] Parse the judge's response into a structured JudgeResult
- [ ] Add `judge` correctness config field with criteria strings
- [ ] Integrate into the verifier pipeline as the final step

## Acceptance Criteria

- Judge verifier sends conversation context and criteria to an LLM and gets a pass/fail verdict
- Judge reasoning is captured and included in the result
- Judge is only invoked when correctness.judge criteria are defined
- Cost of judge invocations is tracked separately from the treatment's cost
