---
title: "Add token usage as an optional first-class ranking dimension"
id: "01kzx1v1z"
status: completed
priority: medium
type: feature
tags: ["ranking", "tokens", "reporting"]
created: "2026-08-13"
completed_at: 2026-08-13
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

## Decision

**Chosen: Option A — an independent `tokens` weight** (confirmed by the user on
2026-08-13, over the task's Option B recommendation).

`ranking.weights.tokens` becomes a first-class weight alongside
`correctness`/`cost`/`duration`/`quality`, all still required to be `>= 0` and
to sum to `1.0`. It defaults to `0`, so existing suites — which never set it —
rank byte-for-byte identically. A suite that knows no model pricing sets
`cost: 0` and gives the freed weight to `tokens`, ranking on the model-agnostic
token signal instead.

Rationale / handling the double-count concern: cost and tokens are correlated,
so summing a nonzero `cost` and a nonzero `tokens` weight double-counts the same
economic signal. Rather than forbidding that in code, we document it: set one of
the two to `0`. This keeps the config surface minimal (one more weight, no new
enum, no change to the sum-to-1.0 invariant) at the cost of trusting the user
not to double-weight — which the docs call out explicitly.

Token dimension: scored on **median total tokens (input + output)** per variant,
normalized within each eval against the best (lowest) variant via
`ratioLowerBetter`, exactly like cost/duration. Variants with no usage report 0
total tokens and are treated as the best (mirroring how `cost = 0` is handled).

## Tasks

- [x] Decide the config surface (Option A/B/C above) — capture the decision and
      rationale in this task before coding. Default must leave existing suites
      ranking identically.
- [x] Extend config: add the chosen field(s) to `internal/suite/suite.go`
      (`RankingWeights.Tokens`) and mirror in `internal/report/rank.go`
      (`Weights.Tokens`; `DefaultWeights` leaves it 0).
- [x] Update `internal/suite/validate.go` (`validateRankingWeights`) so the new
      field is validated (≥ 0, and included in the sum-to-1.0 check).
- [x] Score tokens in `rank.go`: accumulate each variant's median total tokens
      per eval (`tokenNormSum`/`tokenMedSum` in `variantAccumulator` /
      `scoreEval` / `foldMetrics`, via `totalTokens`) and fold a
      `ratioLowerBetter`-normalized token term into the composite. (Accumulation
      machinery moved to `rankaccumulate.go` to keep `rank.go` under the 300-line
      limit.)
- [x] Surface the choice in reports: token weight noted in the HTML
      `weightsNote`; `MEDIAN TOKENS` column in markdown and a `median tokens`
      metric/bar in HTML, plus `median_total_tokens` in JSON — all gated on
      `weights.tokens > 0`, so default output is byte-for-byte unchanged.
- [x] Handle the zero-token case: variants with no usage report 0 total tokens
      and are treated as the best via `ratioLowerBetter`, mirroring `cost = 0`;
      no NaN/Inf. Covered by tests.
- [x] Tests: `rank_tokens_test.go` (ordering, unchanged default, zero-token
      degradation, all-zero fallback, median total reported), `tokens_report_test.go`
      (markdown/JSON/HTML gating), plus validation + YAML-parse tests in the
      suite package.
- [x] Update docs (`docs/cli.md`, `docs/getting-started.md`,
      `docs/configuration.md`) to document the new `tokens` weight and explain
      why cost and tokens must not both be weighted.

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
