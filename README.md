# skival

A Go CLI for evaluating AI coding skill performance. Measures **time to completion**, **token usage**, **dollar cost**, and **correctness** across configurable eval suites.

Define a control case and N treatment variations, then compare them head-to-head with statistical rigor.

## Features

- **Configurable eval suites** — YAML-based definitions for prompts, correctness criteria, and environment setup
- **Multi-treatment comparison** — Run a control and N treatments side-by-side, rank by weighted score
- **Multi-sample runs** — Run each treatment multiple times for statistical confidence (median, CV)
- **Correctness verification** — Pluggable verifiers: substring matching, script execution, HTTP state checks, LLM judge
- **Structured reporting** — Markdown and JSON output with per-eval breakdowns and aggregate rankings
- **Extensible runners** — Built on [agentrunner](https://github.com/driangle/agentrunner-go) for Claude Code CLI, designed to support other coding assistants in the future

## Project Structure

```
skival/
  apps/
    cli/              # Go CLI entry point
  docs/
    PLAN.md           # High-level architecture and implementation plan
  tasks/              # Task tracking
```

## Quick Start

```bash
# Define your eval suite
cat > suite.yaml <<EOF
version: 1
description: "My first eval suite"
evals:
  - id: hello-world
    prompt: "Create a hello world program in Go"
    correctness:
      expected_output: ["Hello, world!"]
    treatments:
      control:
        name: "baseline"
      variations:
        - name: "with-skill"
          skill: "./skills/my-skill"
EOF

# Run the eval
skival run suite.yaml --samples 3 --results-dir ./results
```

## Usage

```
skival run <suite.yaml>        Run an eval suite
skival report <results-dir>    Generate reports from saved results
```

### Key Flags

| Flag | Description |
|------|-------------|
| `--samples N` | Number of runs per treatment (default: 1) |
| `--results-dir` | Directory for results output |
| `--treatments` | Filter to specific treatments |
| `--evals` | Filter to specific eval IDs |
| `--format` | Output format: `markdown`, `json` (default: `markdown`) |

## Configuration

See [docs/PLAN.md](docs/PLAN.md) for the full configuration schema and architecture.

## License

MIT
