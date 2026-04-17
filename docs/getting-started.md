# Getting Started

## Installation

Build from source (requires Go 1.22+):

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
      execute: true
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

You'll see a markdown report printed to stdout with results for each treatment.

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

## What's Next?

- [Configuration](/configuration) -- Full suite.yaml reference
- [CLI](/cli) -- All commands and flags
- [Verifiers](/verifiers) -- Correctness verification strategies
