---
id: "01kpj1s17"
title: "Replace treatments control/variations with flat variants list"
status: completed
priority: high
dependencies: []
tags: ["yaml-api"]
created_at: 2026-04-19
completed_at: 2026-04-19
---

# Replace treatments control/variations with flat variants list

## Objective

Replace the `treatments` object (with its `control` + `variations` split) with a flat `variants` list. The control/variation distinction adds no runtime value — `IsControl` is only used as a JSON output label. The first variant can be treated as the baseline/control internally.

### Current shape

```yaml
treatments:
  control:
    name: baseline
  variations:
    - name: gpt-5-tools
    - name: claude-code
```

### Target shape

```yaml
variants:
  - name: baseline
  - name: gpt-5-tools
  - name: claude-code
```

First variant is the control for ranking/reporting purposes.

## Tasks

- [x] Add `Variants []Treatment` field to `Eval` in `internal/suite/suite.go`
- [x] Migrate `treatments` → `variants` in loader (control becomes first element, variations follow)
- [x] Update `collectTreatments` in executor to iterate `Variants` (first entry gets `IsControl: true`)
- [x] Update `resolveRunnerConfig` to iterate `Variants` instead of `Control` + `Variations`
- [x] Update validation to check all variants (runner, model, prompt, skill/skills, config_dir)
- [x] Update matrix expansion to produce `Variants` instead of `Control` + `Variations`
- [x] Update `resolvePaths` / `resolveTreatmentPaths` for variants
- [x] Update `migrateAllowedTools` for variants
- [x] Update `apps/cli/cmd/validate.go` display
- [x] Update all example suite.yaml files to use `variants`
- [x] Update all tests (loader, validate, executor, report, persist)
- [x] Ensure `TestLoad_Examples` passes

## Acceptance Criteria

- `variants` is a flat list — no control/variations split in YAML
- First variant is treated as control for reporting (`IsControl: true` in JSON output)
- Old `treatments` YAML still loads via migration with deprecation warning
- Matrix expansion produces a flat `Variants` list
- All tests pass
