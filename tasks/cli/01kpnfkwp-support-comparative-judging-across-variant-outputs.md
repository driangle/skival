---
title: "Support comparative judging across variant outputs"
id: "01kpnfkwp"
status: pending
priority: medium
effort: large
type: feature
tags: ["verifier", "judge", "ranking"]
created: "2026-04-20"
---

# Support comparative judging across variant outputs

## Objective

Extend the judge verifier beyond per-run PASS/FAIL so it can compare the outputs of multiple variants on the same eval and produce a signal that feeds into ranking. Today each run is judged in isolation; ties between variants that all pass are broken only by cost/duration. A comparative judge can surface quality differences that absolute grading misses and give users a richer ranking axis.

## Open questions (must be answered before implementation)

These shape the design — resolve them before writing code.

1. **Scope of comparison.** Across *variants* on the same eval (most useful for ranking), across *runs of the same variant* (stability), or both? Default assumption: across variants.
2. **Comparison style.** Pick one:
   - Pairwise winner selection (Bradley-Terry / Elo aggregation) — O(V²) judge calls per eval.
   - N-way ranked list in a single judge call — 1 call per eval, but context-heavy and position-biased.
   - Rubric-scored (judge scores each output 1–5 per criterion).
3. **When does it run?** Always, or only as a tiebreaker among variants that all passed their per-run judges? The tiebreaker option is cheaper and preserves correctness-first ranking.
4. **How does it feed `RankVariants`?** Add a new `QualityScore` dimension + weight in `report.Weights`, replace an existing dimension, or emit a separate "comparison report" alongside the existing rank?
5. **Per-eval vs. suite-aggregate.** Are comparative scores aggregated across all evals (like pass rate), or reported per-eval with a summary roll-up?
6. **Configuration surface.** How do users opt in via `suite.yaml`? A new top-level `compare:` block with criteria + model, per-eval `compare:` overrides, or extend the existing `verify:` list with a new comparative step type?
7. **Bias mitigation.** Do we require swap-and-retry (ask A-vs-B and B-vs-A) for pairwise? Shuffle order for N-way? This doubles cost but is standard practice.
8. **Output handling.** If variant outputs are long (multi-KB), do we pass them verbatim, truncate, or summarize before comparing? Affects judge context cost and fidelity.
9. **Cost controls.** Hard cap on judge calls per eval? Opt-in flag (`--compare`) so the default run stays cheap?
10. **Failure semantics.** What happens when the comparative judge errors or returns unparseable output — skip that eval, fall back to pass/fail only, or fail the whole suite?

## Tasks

- [ ] Resolve the open questions above with the user; record decisions in the task body before proceeding.
- [ ] Design the `suite.yaml` surface for opt-in comparative judging (criteria, model, per-eval vs. suite-level).
- [ ] Add a `ComparativeJudge` type in `internal/verifier/` that takes N variant outputs for an eval and returns a per-variant score or ranking.
- [ ] Wire the comparative judge into the executor so it runs after per-run verification completes for all variants of an eval.
- [ ] Extend `result.SuiteResult` (or a sibling structure) to carry comparative scores without breaking existing persistence.
- [ ] Update `internal/report/rank.go` — add a `QualityScore` field to `VariantRank` and a corresponding weight to `Weights`, with backwards-compatible defaults.
- [ ] Surface comparative results in the HTML, Markdown, and JSON reports.
- [ ] Tests: unit tests for the comparative judge (mocked runner), integration test via an example suite, ranking tests covering the new weight.
- [ ] Document the new suite.yaml fields and CLI flag in `docs/`.

## Acceptance Criteria

- Users can opt in to comparative judging through `suite.yaml` without affecting existing suites.
- The judge compares variant outputs for a given eval and produces a per-variant score or rank.
- Comparative scores are persisted in the suite result and visible in all three report formats.
- `RankVariants` incorporates the new signal via a configurable weight; default weights preserve today's ranking behavior when the feature is unused.
- Bias mitigation strategy (swap/shuffle or documented rationale for skipping it) is implemented.
- Judge-call cost is bounded and documented; failures degrade gracefully to per-run pass/fail.
- New and existing tests pass; `taskmd validate` passes.
