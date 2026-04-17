---
title: "Merge runner and runner_config in suite loader"
id: "01kpcvjj5"
status: pending
priority: critical
type: feature
effort: small
tags: ["backend", "suite"]
dependencies: ["01kpcvhw5"]
created: "2026-04-17"
---

# Merge runner and runner_config in suite loader

## Objective

Extend `mergeDefaults` in the suite loader to propagate `runner` and `runner_config` from defaults → eval → treatment, with deep-merge semantics for `runner_config` maps. This ensures treatments inherit runner settings from their parent eval and suite defaults.

## Tasks

- [ ] Add a `mergeMaps(base, override map[string]any) map[string]any` helper that deep-merges two maps (override keys win)
- [ ] Extend `mergeDefaults` to propagate `Runner` from defaults to evals when the eval's runner is empty
- [ ] Extend `mergeDefaults` to deep-merge `RunnerConfig` from defaults into each eval
- [ ] Add a `resolveRunnerConfig` step that propagates `Runner` and deep-merges `RunnerConfig` from eval into each treatment (control + variations)
- [ ] Wire `resolveRunnerConfig` into the loader pipeline after `mergeDefaults`

## Acceptance Criteria

- [ ] A treatment with no `runner` inherits from its eval, which inherits from defaults
- [ ] `runner_config` is deep-merged: treatment keys override eval keys which override default keys
- [ ] A treatment that explicitly sets `runner` is not overwritten by eval or defaults
- [ ] Unit tests cover the full merge chain (defaults → eval → treatment) for both `runner` and `runner_config`
- [ ] All existing tests pass
