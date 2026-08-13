# Getting Started

## Installation

### Homebrew

```bash
brew install driangle/tap/skival
```

### From source

Requires Go 1.22+:

```bash
git clone https://github.com/driangle/skival.git
cd skival
make install
```

This installs `skival` to your `$GOPATH/bin`.

## Your First Suite

Create a file called `suite.yaml`:

```yaml
version: 1
description: "My first eval suite"
defaults:
  runner: claude-code
evals:
  - id: hello-world
    prompt: "Create a file called hello.txt containing 'Hello, World!'"
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
      - type: check
        run: "cat hello.txt | grep 'Hello, World!'"
    variants:
      - name: baseline
      - name: with-skill
        skill: "./skills/my-skill.md"
```

This suite defines a single eval with two variants: a baseline and a variant that injects a skill.

## Running the Suite

```bash
skival run suite.yaml
```

You'll see a markdown report printed to stdout with results for each variant:

```
# Eval Report

My first eval suite

**Started:** 2025-07-10 14:32:01
**Finished:** 2025-07-10 14:33:48

## Results

EVAL         VARIANT     SAMPLE  STATUS  COST     DURATION  TOKENS (IN/OUT)
----         -------     ------  ------  ----     --------  ---------------
hello-world  baseline    1       pass    $0.0042  12.3s     18.2k/1.1k
hello-world  with-skill  1       pass    $0.0038  9.1s      16.4k/980

## Rankings

RANK  VARIANT     SCORE  PASS RATE  MEDIAN COST  MEDIAN DURATION
----  -------     -----  ---------  -----------  ---------------
#1    with-skill  0.872  100%       $0.0038      9.1s
#2    baseline    0.811  100%       $0.0042      12.3s
```

The composite `SCORE` weights correctness, cost, duration, and (optionally) tokens and quality. If you don't know a model's pricing — local models, self-hosted models, or the `exec` runner all report `$0.0000` — rank on token usage instead of cost by moving the economic weight to `tokens`; a `MEDIAN TOKENS` column then joins the rankings table. See [Configuration — Ranking on tokens instead of cost](/configuration#ranking-on-tokens-instead-of-cost).

With `--samples 3`, you get aggregate statistics per variant:

```
## Results

EVAL         VARIANT     SAMPLE  STATUS  COST     DURATION  TOKENS (IN/OUT)
----         -------     ------  ------  ----     --------  ---------------
hello-world  baseline    1       pass    $0.0042  12.3s     18.2k/1.1k
hello-world  baseline    2       pass    $0.0039  11.8s     17.9k/1.0k
hello-world  baseline    3       pass    $0.0045  13.1s     18.6k/1.2k
hello-world  baseline    agg     PASS    $0.0042 [$0.0039–$0.0045]  12.3s [11.8s–13.1s] cost_cv=7.1% dur_cv=5.3%  18.2k/1.1k
hello-world  with-skill  1       pass    $0.0038  9.1s      16.4k/980
hello-world  with-skill  2       pass    $0.0035  8.7s      16.1k/940
hello-world  with-skill  3       pass    $0.0040  9.8s      16.8k/1.0k
hello-world  with-skill  agg     PASS    $0.0038 [$0.0035–$0.0040]  9.1s [8.7s–9.8s] cost_cv=6.6% dur_cv=6.1%  16.4k/980
```

The `agg` row reports descriptive statistics only — the median, the min–max
range in brackets, the coefficient of variation (`cost_cv` / `dur_cv`) as a
measure of run-to-run spread, and the median input/output token usage. These summarize the samples you collected; they
are not confidence intervals or significance tests. The coefficient of variation
requires at least 3 samples to be meaningful, so it is omitted for smaller runs.

You can also output results as JSON with `--format json` for programmatic consumption.

### Save Results

```bash
skival run suite.yaml --results-dir ./results
```

Results are persisted to disk so you can regenerate reports later:

```bash
skival report ./results
```

### Multiple Samples

To see how much a variant's results vary from run to run, sample each variant multiple times:

```bash
skival run suite.yaml --samples 3 --results-dir ./results
```

This gives you median values and coefficient of variation for cost, duration, and correctness.

### Retrying Flaky Runs

If your evals are sensitive to transient failures (timeouts, network errors), add retry configuration:

```yaml
defaults:
  retry:
    max_attempts: 3
    backoff: exponential
    delay: 2s
```

This retries transient failures up to 2 additional times with exponential backoff. Set `on: all` to also retry correctness failures. See [Configuration — Retry](/configuration#retry) for the full reference.

## What's Next?

- [Configuration](/configuration) -- Full suite.yaml reference
- [CLI](/cli) -- All commands and flags
- [Verifiers](/verifiers) -- Correctness verification strategies
- [Examples](/examples) -- Complete example suites for every feature
