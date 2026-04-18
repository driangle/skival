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
evals:
  - id: hello-world
    prompt: "Create a file called hello.txt containing 'Hello, World!'"
    model: "claude-sonnet-4-6"
    correctness:
      agent_exits_ok: true
      script: "cat hello.txt | grep 'Hello, World!'"
    treatments:
      control:
        name: baseline
      variations:
        - name: with-skill
          skill: "./skills/my-skill.md"
```

This suite defines a single eval with two treatments: a baseline control and a variation that injects a skill.

## Running the Suite

```bash
skival run suite.yaml
```

You'll see a markdown report printed to stdout with results for each treatment:

```
# Eval Report

My first eval suite

**Started:** 2025-07-10 14:32:01
**Finished:** 2025-07-10 14:33:48

## Results

EVAL         TREATMENT   SAMPLE  STATUS  COST     DURATION
----         ---------   ------  ------  ----     --------
hello-world  baseline    1       pass    $0.0042  12.3s
hello-world  with-skill  1       pass    $0.0038  9.1s

## Rankings

RANK  TREATMENT   SCORE  PASS RATE  MEDIAN COST  MEDIAN DURATION
----  ---------   -----  ---------  -----------  ---------------
#1    with-skill  0.872  100%       $0.0038      9.1s
#2    baseline    0.811  100%       $0.0042      12.3s
```

With `--samples 3`, you get aggregate statistics per treatment:

```
## Results

EVAL         TREATMENT   SAMPLE  STATUS  COST     DURATION
----         ---------   ------  ------  ----     --------
hello-world  baseline    1       pass    $0.0042  12.3s
hello-world  baseline    2       pass    $0.0039  11.8s
hello-world  baseline    3       pass    $0.0045  13.1s
hello-world  baseline    agg     PASS    $0.0042 [$0.0039–$0.0045]  12.3s [11.8s–13.1s] cost_cv=7.1% dur_cv=5.3%
hello-world  with-skill  1       pass    $0.0038  9.1s
hello-world  with-skill  2       pass    $0.0035  8.7s
hello-world  with-skill  3       pass    $0.0040  9.8s
hello-world  with-skill  agg     PASS    $0.0038 [$0.0035–$0.0040]  9.1s [8.7s–9.8s] cost_cv=6.6% dur_cv=6.1%
```

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

For statistical confidence, run each treatment multiple times:

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
