---
id: "01m3pbbc2"
title: "Separate infrastructure failure from correctness failure"
status: pending
priority: critical
effort: medium
type: bug
tags: ["verifier", "executor", "measurement"]
created: 2026-08-12
context: ["internal/verifier/judge.go", "internal/executor/retry.go", "internal/result/result.go"]
verify:
  - type: bash
    run: "go test ./internal/verifier/... ./internal/executor/..."
  - type: assert
    check: "A judge invocation error is not recorded as Pass=false and is retried under retry.on=transient"
---

# Separate infrastructure failure from correctness failure

## Objective

`JudgeVerifier.Verify` returns `Pass: false` when the judge *invocation* fails
(`internal/verifier/judge.go:56`). A rate limit, a network blip, or a
misconfigured judge is therefore recorded as "the agent produced a wrong
answer", feeds the pass rate, and feeds the ranking.

Reproduced with an `exec` variant, where the judge is structurally unable to run
because it inherits the variant's runner with no `runner_config`:

```
t > mine > verify judge: FAIL (judge invocation failed: exec runner: missing config)
t     mine    1    fail    $0.0000  0ms
```

The agent answered correctly. It is scored as a correctness failure.

This also defeats retries: `shouldRetry` only retries when `run.Err != nil`, and
a judge failure sets `Pass=false` with no `Err`, so the default
`retry.on: transient` policy will never retry a transient judge outage.

For a tool whose headline output is a pass rate, this is the most damaging
defect in the codebase.

## Tasks

- [ ] Introduce an explicit `Errored` / indeterminate outcome on `VerifyResult`
      distinct from `Pass: false`
- [ ] Propagate it to `RunResult` so aggregation and ranking can exclude
      indeterminate runs rather than counting them as failures
- [ ] Make `shouldRetry` treat verifier infrastructure errors as transient
- [ ] Surface indeterminate runs distinctly in reports (not silently dropped)
- [ ] Reject `judge` verify steps on runners that cannot host a judge, at
      validation time rather than at run time

## Acceptance Criteria

- A judge whose invocation fails does not lower the pass rate
- Such a run is retried under `retry.on: transient`
- Reports distinguish "failed" from "could not be determined"
