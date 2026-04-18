# skival

[![CI](https://github.com/driangle/skival/actions/workflows/ci.yaml/badge.svg)](https://github.com/driangle/skival/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/driangle/skival/branch/main/graph/badge.svg)](https://codecov.io/gh/driangle/skival)

A Go CLI for evaluating AI coding skill performance. Measures **time to completion**, **token usage**, **dollar cost**, and **correctness** across configurable eval suites.

Define a control case and N treatment variations, then compare them head-to-head with statistical rigor.

**[Documentation](https://driangle.github.io/skival/)**

## Features

- **Configurable eval suites** — YAML-based definitions for prompts, correctness criteria, and environment setup
- **Multi-treatment comparison** — Run a control and N treatments side-by-side, rank by weighted composite score
- **Multi-sample runs** — Run each treatment multiple times for statistical confidence (median, CV)
- **Matrix syntax** — Define dimensions (e.g. runner × model) and auto-generate a Cartesian product of treatments
- **Per-treatment overrides** — Customize prompt, model, runner, skills, env vars, config directory, and allowed tools per treatment
- **Skill injection** — Inject single or multiple skill files into agent system prompts for A/B testing skill effectiveness
- **Working directory isolation** — Optionally copy the eval directory per sample to prevent cross-sample state pollution
- **Setup lifecycle hooks** — Run shell commands before, between (reset), and after samples for fixture management
- **Correctness verification** — Pluggable verifier pipeline: exit code, substring matching, custom scripts, HTTP state checks, LLM judge
- **Multi-runner support** — Built on [agentrunner](https://github.com/driangle/agentrunner) with support for Claude Code, Ollama, Codex, and Aider
- **External eval files** — Reference eval definitions from separate YAML files for reuse across suites
- **Structured reporting** — Markdown and JSON output with per-eval breakdowns, aggregate metrics, and ranked treatments
- **Suite validation** — Validate suite YAML structure and required fields without executing

## Quick Start

```bash
# Define your eval suite
cat > suite.yaml <<EOF
version: 1
description: "My first eval suite"
evals:
  - id: hello-world
    prompt: "Create a hello world program in Go"
    model: "claude-sonnet-4-6"
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
skival validate <suite.yaml>   Validate suite structure without executing
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
| `-v, --verbose` | Enable debug-level logging |

## Configuration

See the [documentation site](https://driangle.github.io/skival/) for the full configuration schema, verifier reference, and CLI guide.

## License

MIT
