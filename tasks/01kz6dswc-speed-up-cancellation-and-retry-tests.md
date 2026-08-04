---
id: "01kz6dswc"
title: "Speed up cancellation and retry tests with injected time"
status: pending
priority: low
effort: small
type: chore
tags: ["testing", "tech-debt"]
created: 2026-08-04
---

# Speed up cancellation and retry tests with injected time

## Objective

The test suite spends most of its wall-clock in a handful of tests that sleep on
real time. In `internal/verifier`, context-cancellation tests burn fixed real
delays:

```
--- PASS: TestCommandProbeVerifier_ContextCancellation (10.00s)
--- PASS: TestCheckOutputVerifier_RespectsContextCancellation (10.00s)
--- PASS: TestStateVerifier_RespectsContextCancellation (5.01s)
--- PASS: TestHTTPProbeVerifier_RespectsContextCancellation (5.00s)
```

Executor/retry tests also use real `time.Sleep` backoffs (~5.7s for the package).
A cancellation test should prove *fast* cancellation, not wait 10s to observe it.
The whole suite runs ~35s when it could be sub-second.

## Tasks

- [ ] Rework verifier cancellation tests to cancel immediately and assert prompt
      return, instead of racing a long real sleep
- [ ] Inject a clock / make backoff delays overridable so `retry` tests don't
      sleep real seconds (`internal/executor/retry.go`, `runSample` backoff)
- [ ] Ensure any remaining real-time waits use short, bounded deadlines
- [ ] Confirm total `go test ./...` wall-clock drops substantially

## Acceptance Criteria

- No single test spends >1s in a fixed real-time sleep to prove cancellation
- `go test ./...` still passes and runs materially faster
