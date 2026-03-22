---
title: "Persist trial conversations (agent + judge)"
id: "01km8d1bn"
status: completed
priority: low
type: feature
tags: ["persistence"]
created: "2026-03-21"
---

# Persist trial conversations (agent + judge)

## Objective

Per the plan (`docs/PLAN.md`), each trial (eval × treatment × sample) should persist conversation history alongside its metrics JSON. This enables post-hoc analysis of agent behavior, debugging failed runs, and comparing conversation patterns across treatments.

Two distinct conversations need to be captured per trial:
1. **Agent conversation** — the main agentrunner interaction (already exposed via `Result`)
2. **Judge conversation** — the LLM judge verification call (currently discarded in `JudgeVerifier.Verify`)

## Terminology

A **Trial** is the unique combination of `(eval_id, treatment_name, sample_number)` — the atomic unit of execution. This maps to what is currently called `RunResult` in the code. The persist layer already encodes this as `evals/{eval_id}/{treatment_name}/run-{N}.json`.

## Tasks

- [x] Capture agent conversation history from agentrunner `Result` into `RunResult` (add a `Conversation` field)
- [x] Extend `VerifyResult` (or `StepResult`) to carry an optional conversation from verifier steps
- [x] In `JudgeVerifier.Verify`, capture and return the judge's conversation from the runner `Result`
- [x] Thread the judge conversation from the pipeline result back into `RunResult`
- [x] In `persist`, write `run-N.conversation.jsonl` for the agent conversation (one JSON object per message)
- [x] In `persist`, write `run-N.judge.jsonl` for the judge conversation when present
- [x] Update `persist.Load` to optionally load conversation files when available
- [x] Add tests for conversation persistence (write + read round-trip, both agent and judge)

## Acceptance Criteria

- After a suite run with `--results-dir`, each treatment directory contains `run-N.conversation.jsonl` files for the agent conversation
- When a judge verifier ran, a `run-N.judge.jsonl` file is also present
- Each `.jsonl` file contains one JSON object per line representing a conversation message
- Trials where the runner errors out do not produce conversation files
- Trials with no judge criteria do not produce `.judge.jsonl` files
- Loading results via `persist.Load` does not fail when conversation files are absent (backwards compatible)
