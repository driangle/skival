---
id: "01kpj1nss"
title: "Move judge_model into judge verify step as model field"
status: completed
priority: medium
dependencies: []
tags: ["yaml-api", "verify"]
created_at: 2026-04-19
completed_at: 2026-04-19
---

# Move judge_model into judge verify step as model field

## Objective

`judge_model` currently lives on `Eval` (and `Defaults`) as a top-level field, separate from the judge verify step that actually uses it. Move it into the judge step as `model` so the configuration is colocated with the step. `defaults.judge_model` should apply to any judge step that doesn't set its own `model`.

### Target YAML shape

```yaml
defaults:
  judge_model: "claude-opus-4-6"   # applied to judge steps missing model

evals:
  - id: example
    verify:
      - type: judge
        criteria: ["Is it correct?"]
        model: "claude-opus-4-6"   # per-step override
```

## Tasks

- [x] Add `Model` field (`yaml:"model,omitempty"`) to `VerifyStep` in `internal/suite/suite.go`
- [x] Remove `JudgeModel` field from `Eval` struct
- [x] Update `mergeDefaults` to apply `defaults.judge_model` into judge verify steps that lack a `model`
- [x] Update `migrateCorrectnessToVerify` to put `Correctness.JudgeModel` on the judge step's `Model` field
- [x] Update `BuildPipeline` to read model from the judge step instead of `pipelineConfig`
- [x] Simplify `WithJudge` pipeline option (no longer needs judgeModel param)
- [x] Update executor to stop passing `eval.JudgeModel` — pipeline reads it from the step
- [x] Update judge_model warning in `validate.go` to check judge steps directly
- [x] Update tests (loader, pipeline, judge, validate)
- [x] Ensure `TestLoad_Examples` passes

## Acceptance Criteria

- Judge model is configured per-step via `model` field on judge verify steps
- `defaults.judge_model` propagates into judge steps that don't set `model`
- `JudgeModel` no longer exists on `Eval`
- All tests pass
