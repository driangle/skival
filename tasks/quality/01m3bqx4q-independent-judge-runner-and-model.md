---
id: "01m3bqx4q"
title: "Configure the judge independently of the variant under test"
status: pending
priority: high
effort: medium
type: bug
tags: ["verifier", "judge", "measurement"]
created: 2026-08-12
context: ["internal/executor/run.go", "internal/verifier/judge.go", "internal/executor/comparative.go"]
verify:
  - type: bash
    run: "go test ./internal/verifier/... ./internal/executor/..."
  - type: assert
    check: "Every variant in a multi-runner suite is graded by the same judge runner and model"
---

# Configure the judge independently of the variant under test

## Objective

`buildSamplePipeline` (`internal/executor/run.go:90`) hands the judge the
*variant's own runner*, and `DefaultJudgeModel` is the hardcoded string
`"claude-haiku-4-5-20251001"`. Three consequences:

1. In a matrix suite, each variant is graded by a different judge — which makes
   cross-variant correctness non-comparable, the exact comparison skival exists
   to make.
2. An `ollama` variant sends a Claude model ID to Ollama.
3. An `exec` variant cannot be judged at all: the judge inherits the exec runner
   with no `runner_config` and fails with "missing config".

Hardcoding a vendor model as the default also sits directly against CLAUDE.md's
rule that the CLI must not assume the user's stack.

Comparative judging has the same shape: `runComparison` picks
`candidates[0].runner`, i.e. whichever variant happened to pass first.

## Tasks

- [ ] Add a suite-level `judge:` block with its own `runner`, `model`, and
      `runner_config`
- [ ] Resolve the judge once per suite and share it across per-run judging and
      comparative judging
- [ ] Require an explicit judge configuration instead of defaulting to a vendor
      model; fail validation with a clear message when a `judge` step is used
      without one
- [ ] Record the judge runner and model in results and reports

## Acceptance Criteria

- All variants in a suite are graded by the same judge
- An exec-only suite can use a `judge` verify step
- No vendor model ID is hardcoded as a default
