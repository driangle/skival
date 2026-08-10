---
title: "Support comparative judging across variant outputs"
id: "01kpnfkwp"
status: completed
priority: medium
effort: large
type: feature
tags: ["verifier", "judge", "ranking"]
created: "2026-04-20"
completed_at: 2026-08-10
---

# Support comparative judging across variant outputs

## Objective

Extend the judge verifier beyond per-run PASS/FAIL so it can compare the outputs of multiple variants on the same eval and produce a signal that feeds into ranking. Today each run is judged in isolation; ties between variants that all pass are broken only by cost/duration. A comparative judge can surface quality differences that absolute grading misses and give users a richer ranking axis.

## Resolved decisions (2026-08-10)

Approved by the user before implementation:

1. **Scope** — Across *variants* on the same eval. (Not across runs of the same variant.)
2. **Comparison style** — **N-way scored, 1 judge call per eval.** The judge sees all passing variant outputs for an eval and returns a 1–5 quality score per variant. Maps to `[0,1]` (score/5) for ranking. Order is shuffled to mitigate position bias.
3. **When it runs** — **Tiebreaker only.** Runs only among variants that all passed their per-run judges, preserving correctness-first ranking. Skipped when fewer than 2 variants passed.
4. **Ranking integration** — **Weighted `QualityScore` dimension.** Add `Quality float64` to `VariantRank` and a `Quality` weight to `Weights`, defaulting to **0** so existing suites rank byte-identically. When `compare:` is enabled, the CLI applies a non-zero quality weight (renormalizing the others). Also surface the raw comparison as its own report section.
5. **Aggregation** — Per-eval scores **and** a suite-level roll-up (mean of per-eval quality) feeding the ranking.
6. **Config surface** — **New top-level `compare:` block** (criteria + model + enable) with optional per-eval `compare:` overrides. Keeps cross-variant lifecycle separate from the per-run `verify:` list.
7. **Bias mitigation** — Shuffle variant order before each N-way judge call.
8. **Long outputs** — Truncate each variant output to a configurable char cap before judging. (Revisit summarize-then-compare only if truncation proves to bias results.)
9. **Cost controls** — Opt-in via config; `--compare`/`--no-compare` CLI flag overrides; one judge call per eval (bounded by design).
10. **Failure semantics** — Degrade gracefully: on judge error or unparseable output, skip comparative scoring for that eval and fall back to per-run pass/fail. Never fail the whole suite.

## Tasks

- [x] Resolve the open questions above with the user; record decisions in the task body before proceeding.
- [x] Design the `suite.yaml` surface for opt-in comparative judging (criteria, model, per-eval vs. suite-level).
- [x] Add a `ComparativeJudge` type in `internal/verifier/` that takes N variant outputs for an eval and returns a per-variant score or ranking.
- [x] Wire the comparative judge into the executor so it runs after per-run verification completes for all variants of an eval.
- [x] Extend `result.SuiteResult` (or a sibling structure) to carry comparative scores without breaking existing persistence.
- [x] Update `internal/report/rank.go` — add a `QualityScore` field to `VariantRank` and a corresponding weight to `Weights`, with backwards-compatible defaults.
- [x] Surface comparative results in the HTML, Markdown, and JSON reports.
- [x] Tests: unit tests for the comparative judge (mocked runner), integration test via an example suite, ranking tests covering the new weight.
- [x] Document the new suite.yaml fields and CLI flag in `docs/`.

## Acceptance Criteria

- Users can opt in to comparative judging through `suite.yaml` without affecting existing suites.
- The judge compares variant outputs for a given eval and produces a per-variant score or rank.
- Comparative scores are persisted in the suite result and visible in all three report formats.
- `RankVariants` incorporates the new signal via a configurable weight; default weights preserve today's ranking behavior when the feature is unused.
- Bias mitigation strategy (swap/shuffle or documented rationale for skipping it) is implemented.
- Judge-call cost is bounded and documented; failures degrade gracefully to per-run pass/fail.
- New and existing tests pass; `taskmd validate` passes.
