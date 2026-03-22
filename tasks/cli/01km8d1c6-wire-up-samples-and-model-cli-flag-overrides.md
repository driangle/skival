---
title: "Wire up --samples and --model CLI flag overrides"
id: "01km8d1c6"
status: completed
priority: low
type: chore
tags: ["cli"]
created: "2026-03-21"
---

# Wire up --samples and --model CLI flag overrides

## Description

The `skival run` command defines `--samples` and `--model` flags but never reads or applies them. They are dead code. These flags should override the suite/eval-level defaults so users can quickly re-run with different sample counts or models without editing YAML.

## Tasks

- [x] Read `--samples` flag in `run.go` and pass it to `executor.Options`
- [x] Add `Samples` and `Model` fields to `executor.Options`
- [x] In `executeTreatment`, use `Options.Samples` to override `eval.Samples` when set
- [x] In `buildRunOptions`, use `Options.Model` to override eval/treatment model when set
- [x] Add tests verifying CLI flag overrides take precedence over YAML values
- [x] Remove or update the flag definitions if the semantics change

## Acceptance Criteria

- `skival run suite.yaml --samples 5` runs 5 samples per treatment regardless of YAML `samples` value
- `skival run suite.yaml --model claude-haiku-4-5-20251001` overrides the model for all treatments
- When flags are not provided, behavior is unchanged (YAML values apply)
- Existing tests continue to pass
