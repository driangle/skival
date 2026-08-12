# skival — Differentiation Strategy

> Internal strategy document. Excluded from the published docs site
> (`srcExclude: specs/**`). Companion to the task files under `tasks/`.

## Situation

skival is not alone in its problem space, and the incumbents are strong.

| Tool | What it owns | Shape |
| --- | --- | --- |
| [promptfoo](https://github.com/promptfoo/promptfoo) | YAML matrix eval of prompts × providers, with assertions. ~10.8k stars, 150k+ OSS users. Acquired by OpenAI (Mar 2026), staying open source. | Node CLI |
| [Inspect AI](https://github.com/UKGovernmentBEIS/inspect_ai) | Research-grade agent evals: sandboxing, solvers/scorers, 200+ prebuilt evals. Used by frontier labs. | Python framework |
| DeepEval | pytest-native metric library, 50+ research-backed metrics. | Python |
| Langfuse / Braintrust / LangSmith | Production tracing and eval over time. | SaaS + SDK |
| SWE-bench, Terminal-Bench | Standard task sets and leaderboards for agent+model pairs. | Benchmarks |

The uncomfortable fact: promptfoo ships a guide titled *Test Agent Skills* whose
core instruction is "keep the model, task files, and permissions the same, swap
only the `SKILL.md`, and compare the results." It has `skill-used` trace
assertions, `cost` and `latency` assertions, `--repeat N`, a web viewer, and CI
exit codes. That is skival's flagship use case, already solved, by a tool with
four orders of magnitude more users.

**Feature parity with promptfoo is not a viable strategy.** Any plan that
amounts to "add the assertions promptfoo has" loses by definition.

## The gap nobody fills

Every tool above answers *"what score did this configuration get?"*

None of them answer *"is this change actually an improvement, or is it noise?"*

That gap is real and it matters, because agent runs are both **high-variance**
and **expensive**:

- promptfoo's `--repeat N` samples repeatedly and averages. It does not model
  the spread, and it will happily report 3/3 vs 2/3 as a difference.
- Inspect AI has epochs and reducers, but leaves the inference to you and costs
  a Python environment plus Docker to adopt.
- Published harness comparisons swing 10–20 points on identical model weights.
  Anyone A/B-testing a prompt or skill on 3 samples is mostly measuring noise.

The person running skival is trying to make a **decision** — ship this skill or
not, switch to this model or not — and the honest answer is frequently "your
sample size cannot support that call." No tool in this space says so.

## Positioning

> **skival is a CI-native A/B test for agent configurations.**
> It tells you whether a change is a real improvement, an actual regression, or
> statistically indistinguishable — on your own tasks, against any agent you can
> invoke from a shell, from a single static binary.

Three assets make this defensible, and they are assets skival already has:

1. **A single static Go binary.** Every competitor needs a Node or Python
   environment, and Inspect AI needs Docker. skival drops into any CI image with
   no runtime. This is the cheapest possible adoption story and it is not
   something a Node-based incumbent can copy.
2. **The `exec` runner.** promptfoo integrates SDK by SDK
   (`anthropic:claude-agent-sdk`, `openai:codex-sdk`, `opencode:sdk`). skival's
   contract is "any program, prompt via stdin/env/argfile, optional JSONL event
   stream" — which can evaluate an internal agent, a competitor's CLI, or a bash
   script, with zero code in skival. Harness-agnosticism *by contract* rather
   than by integration.
3. **Repeated sampling is already in the data model.** Samples, medians, spread,
   and coefficient of variation exist. The statistical layer is a short reach
   from here, and no competitor is reaching for it.

## Non-goals

Stated explicitly so they can be declined without re-litigation:

- **Do not chase provider breadth.** The `exec` runner is the answer to "does it
  support X." Adding first-class Go integrations per vendor is a treadmill
  skival loses.
- **Do not chase assertion-catalogue breadth.** Verifiers should cover what is
  needed to establish correctness on the user's own tasks, not compete with
  DeepEval's 50+ metrics.
- **Do not build red-teaming or safety scanning.** That is promptfoo's other
  half and is now OpenAI-funded.
- **Do not build a hosted service, dashboard, or tracing backend.** Langfuse and
  Braintrust own production observability. skival is a CI-time tool.
- **Do not add a standard task set.** SWE-bench and Terminal-Bench exist. skival
  runs *your* tasks.

## Plan

### Phase 0 — Earn the right to be trusted

A measurement instrument that reports wrong numbers has negative value, and
skival currently reports several. Nothing in Phase 1 means anything until this
is done. Concretely: lint gates have not run in CI since April; a judge
invocation failure is recorded as the agent giving a wrong answer; a variant
that cannot execute scores full marks on cost and duration; the `exec` runner
reports 0ms for every run; and `skival report` produces different rankings than
`skival run` on the same data.

This phase is entirely bug-fixing, and it is the highest-value work in the plan.

### Phase 1 — Decision-grade comparison (the wedge)

Replace "here is a composite score" with "here is a verdict you can act on."

- **Paired, interleaved execution.** Run baseline and variant alternately within
  an eval rather than variant-by-variant, so provider-side drift (load, latency,
  silent model updates) hits both arms equally. This is a design property
  competitors cannot retrofit cheaply, because their execution model is a matrix
  sweep, not a paired trial.
- **Uncertainty on the deltas.** Bootstrap percentile intervals on the pass-rate,
  cost, and duration deltas versus baseline. Not p-values, not significance
  theatre — a resampled interval, which is easy to explain and hard to misuse.
- **A verdict, not a leaderboard.** Per metric: `better`, `worse`, or
  `inconclusive`, with the interval shown. `inconclusive` is a first-class,
  frequently-correct answer and is the single most differentiating thing skival
  can print.
- **Adaptive sampling.** `--until-decided` keeps drawing samples until the
  interval resolves or a cost budget is exhausted, then reports which happened.
  This directly attacks the reason people under-sample: agent runs cost money.
  No competitor does this.
- **Regression gating.** `--fail-on regression` turns the verdict into a CI
  gate, which is where the single-binary story pays off.

The existing weighted composite score survives as an opt-in convenience, clearly
labelled as a heuristic. It stops being the headline. The composite is currently
the least defensible artifact skival produces and is also the only part of the
design that competitors deliberately avoid — that should be read as a signal.

### Phase 2 — Verify the work, not the summary

skival's judge currently sees the agent's final text message. For a tool with
"coding" in its purpose, that grades the agent's self-report rather than its
work.

- **Trace assertions** over the already-captured conversation: `tool_used`,
  `tool_not_used`, `tool_call_count`, `tool_sequence`. This closes the one real
  capability gap versus promptfoo's `skill-used`, using data skival already has.
- **Diff-aware judging.** Show the judge the workspace diff produced by the run,
  not only the final message. Isolation already gives a clean before/after pair
  to diff. This is a genuine quality edge: it is the difference between grading
  what an agent *says* it did and what it *did*.

### Phase 3 — Make harness-agnosticism a proof, not a claim

- **Version the exec event schema** as a published spec, plus `skival conform
  ./my-agent` to check a program against it. A contract you can validate is a
  meaningfully stronger claim than a list of supported SDKs.
- **Ship third-party adapters as configuration, not Go code** — codex CLI,
  aider, opencode, cursor-agent as example `runner_config` blocks. Every adapter
  that requires no code change is direct evidence for the positioning, and it
  costs one YAML file instead of one package.

### Phase 4 — Distribution

Single-binary release, Homebrew tap, and a composite GitHub Action so the CI
story is one `uses:` line. Plus an honest positioning page: *when to use
promptfoo instead*. Naming the cases where a competitor is the better answer is
cheap, verifiable, and buys more credibility than a feature matrix.

## Success criteria

skival is differentiated when all of these are true:

1. Running the same suite twice on an unchanged configuration reports
   `inconclusive`, not a spurious winner.
2. A user can gate a PR on "this prompt change is not a regression" with one
   GitHub Action step and no runtime installed.
3. A third-party agent CLI can be evaluated end to end without a line of Go.
4. The docs can state plainly when promptfoo is the better tool, and the
   remaining cases are ones promptfoo genuinely does not serve.

## Risks

- **The wedge is narrow.** If users do not care about statistical honesty, the
  differentiator does not land, and promptfoo wins on breadth. Mitigation:
  Phase 0 and Phase 2 have standalone value regardless.
- **Statistics invite new overclaiming.** Bootstrap intervals on 5 samples are
  still weak. Every interval must ship with its sample count, and the tool must
  refuse to render a verdict below a minimum n.
- **Sunk-cost pull toward parity.** HTML reports, colored output, and more
  verifier types feel productive and move nothing. They belong after Phase 2, or
  nowhere.
