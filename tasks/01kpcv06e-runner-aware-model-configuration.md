---
title: "Runner-aware model configuration"
id: "01kpcv06e"
status: completed
priority: low
type: feature
tags: ["schema", "models"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Runner-aware model configuration

## Objective

Make model configuration runner-aware so that model identifiers are meaningful across different runners. Currently the `model` field assumes Claude model IDs (e.g., `claude-sonnet-4-20250514`), but with multi-runner support, different runners use different model namespaces (OpenAI's `gpt-4o`, Anthropic's `claude-opus-4`, etc.).

This depends on multi-runner support being implemented first.

## Tasks

- [x] Define how `model` interacts with `runner` — validate that the model is compatible with the selected runner, or treat it as an opaque string passed through
- [x] Update validation to warn (not error) if model doesn't match known patterns for the runner
- [x] Ensure model precedence chain (treatment > eval > defaults) works correctly when different treatments use different runners
- [x] Add reporting metadata to show which runner+model combination was used per treatment
- [x] Add tests for model resolution across runner types

## Acceptance Criteria

- Model field is passed through to the selected runner without modification
- Report output includes both runner and model for each treatment
- Validation produces a warning if a model ID doesn't look valid for the selected runner
- Existing suites with model fields continue to work unchanged
