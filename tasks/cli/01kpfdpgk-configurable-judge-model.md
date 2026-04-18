---
title: "Configurable judge model"
id: "01kpfdpgk"
status: pending
priority: medium
type: feature
tags: ["verifier", "configuration"]
created: "2026-04-18"
---

# Configurable judge model

## Objective

Make the LLM model used by the judge verifier configurable instead of hardcoded to `claude-haiku-4-5-20251001`. Users may want a stronger model (Sonnet/Opus) for nuanced correctness checks, or a cheaper model for simple pass/fail evals.

### Configuration

```yaml
evals:
  - id: example
    correctness:
      judge: ["criterion 1", "criterion 2"]
      judge_model: "claude-sonnet-4-6"   # optional, defaults to claude-haiku-4-5-20251001
```

Can also be set at suite defaults level:

```yaml
defaults:
  judge_model: "claude-sonnet-4-6"
```

## Tasks

- [ ] Add optional `judge_model` field to the `Correctness` struct in suite schema
- [ ] Add optional `judge_model` field to suite defaults
- [ ] Apply standard override precedence: eval-level `judge_model` > suite defaults `judge_model` > hardcoded default
- [ ] Update `JudgeVerifier` in `verifier/judge.go` to accept the model as a parameter instead of using the constant
- [ ] Thread the resolved judge model from suite loader through executor to verifier construction
- [ ] Add validation: if `judge_model` is set, warn if `judge` criteria are not also defined
- [ ] Add tests for judge model config parsing and precedence
- [ ] Add test that `JudgeVerifier` uses the configured model
- [ ] Update documentation (verifiers.md, configuration.md) with `judge_model` field

## Acceptance Criteria

- Setting `judge_model` at eval level uses that model for the judge verifier
- Setting `judge_model` at defaults level applies to all evals without an explicit override
- Omitting `judge_model` everywhere uses the current default (`claude-haiku-4-5-20251001`)
- The model used is visible in judge conversation results for debugging
