---
title: "Add token usage as an optional first-class ranking dimension"
id: "01kzx1v1z"
status: pending
priority: medium
type: feature
tags: ["ranking", "tokens", "reporting"]
created: "2026-08-13"
---

# Add token usage as an optional first-class ranking dimension

## Objective

Let a suite rank variants on **token usage**, not just dollar cost. Today the
composite score (`internal/report/rank.go`) has exactly four dimensions —
correctness, cost, duration, quality (`internal/suite/suite.go` →
`RankingWeights`, weights must be ≥ 0 and sum to 1.0). Token usage is now
persisted and reported (task `01kzs8hp2`) but never scored; it only influences
ranking *indirectly through cost*.

The motivation is that **cost is not always available or meaningful**:

- The user may not know the per-model/agent pricing (local models, the exec
  runner, self-hosted or newly-released models), so the runner reports
  `cost_usd = 0` and the cost dimension silently contributes nothing.
- Token usage is a **model-agnostic efficiency signal** — it's the honest way
  to compare a token-hungry cheap model against a terse expensive one without
  trusting a price table.

## Design tension (resolve before/while implementing)

Cost and tokens are strongly correlated (`cost ≈ tokens × price`). Adding a
naïve independent `tokens` weight **alongside** `cost` double-counts the same
underlying signal — the user flagged this as "a bit odd." So the naïve additive
approach is likely the wrong shape. Options to evaluate:

- **Option A — independent `tokens` weight.** Simplest to wire in, but
  double-counts cost. Only sensible if a suite sets `cost: 0`.
- **Option B — one configurable "economy" basis (recommended).** The economic
  dimension is ranked on **either cost or tokens**, chosen per suite (e.g.
  `ranking.economy: cost | tokens`), never both. Avoids double-counting and
  keeps the composite to one economic term.
- **Option C — auto-fallback.** Use cost when any variant reports nonzero cost,
  otherwise fall back to tokens automatically. Model-agnostic with zero config,
  but less explicit/predictable.

Recommendation: **Option B**, defaulting `economy: cost` so existing suites are
unaffected, with a documented note on why cost+tokens are not summed. Confirm
the exact config surface (a `tokens` weight vs. an `economy` selector) before
building. Whichever option is chosen, ranking on tokens should use **median
total tokens (input + output)** per variant, normalized *within each eval*
relative to the best (lowest) variant, exactly like cost/duration
(`ratioLowerBetter` in `rank.go`).

## Tasks

- [ ] Decide the config surface (Option A/B/C above) — capture the decision and
      rationale in this task before coding. Default must leave existing suites
      ranking identically.
- [ ] Extend config: add the chosen field(s) to `internal/suite/suite.go`
      (`RankingWeights` / `Ranking`) and mirror in `internal/report/rank.go`
      (`Weights`, `DefaultWeights`).
- [ ] Update `internal/suite/validate.go` (`validateRankingWeights`) so the new
      field is validated (≥ 0, and — if it's a weight — included in the
      sum-to-1.0 check; if it's an `economy` selector, validate the enum).
- [ ] Score tokens in `rank.go`: accumulate each variant's median total tokens
      per eval (alongside `costMedSum`/`durMedSum` in `variantAccumulator` /
      `scoreEval` / `foldMetrics`) and fold a `ratioLowerBetter`-normalized
      token term into the composite. Ensure only one economic term contributes
      when Option B is chosen.
- [ ] Surface the choice in reports: show which economy basis drove the ranking
      in the markdown/HTML `weightsNote` / rankings block; optionally add a
      token bar to the HTML rankings.
- [ ] Handle the zero-token case: variants/runners with no usage must not be
      unfairly scored (a suite ranking on tokens where a runner reports none
      should degrade gracefully, mirroring how `cost = 0` is handled today).
- [ ] Tests: token-based ranking orders variants correctly; default config
      reproduces current rankings byte-for-byte; validation accepts/rejects the
      new field; zero-token runners degrade gracefully.
- [ ] Update docs (`docs/cli.md`, `docs/getting-started.md`,
      `docs/configuration.md`) to document the new ranking option and explain
      why cost and tokens are not both summed.

## Acceptance Criteria

- A suite can rank variants on token usage (per the chosen config surface), and
  a suite that knows no model pricing can still get a meaningful ranking.
- Cost and tokens are **not** naïvely double-counted; the chosen design avoids
  summing two correlated economic signals (or this is explicitly justified).
- Default configuration leaves all existing suites ranking exactly as before.
- Ranking weights still validate (≥ 0, sum to 1.0 where applicable) and invalid
  config is rejected with a clear message.
- Reports make it clear which economic basis (cost vs. tokens) drove the ranking.
- New tests cover token-based ranking, the unchanged default, validation, and
  the zero-token degradation path; `make check` passes.
