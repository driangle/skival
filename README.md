# skival

[![CI](https://github.com/driangle/skival/actions/workflows/ci.yaml/badge.svg)](https://github.com/driangle/skival/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/driangle/skival/branch/main/graph/badge.svg)](https://codecov.io/gh/driangle/skival)

A Go CLI for evaluating AI coding skill performance. Measures **time to completion**, **token usage**, **dollar cost**, and **correctness** across configurable eval suites.

Define a baseline and any number of variants, then compare them head-to-head with descriptive statistics across repeated samples.

**[Documentation](https://driangle.github.io/skival/)**

## Features

- **Configurable eval suites** — YAML-based definitions for prompts, correctness criteria, and environment setup
- **Multi-variant comparison** — Run any number of variants side-by-side, rank by weighted composite score
- **Multi-sample runs** — Run each variant multiple times and summarize with descriptive statistics (median, min/max spread, coefficient of variation)
- **Matrix syntax** — Define dimensions (e.g. runner × model) and auto-generate a Cartesian product of variants
- **Per-variant overrides** — Customize prompt, model, runner, skills, env vars, config directory, and allowed tools per variant
- **Skill injection** — Inject single or multiple skill files into agent system prompts for A/B testing skill effectiveness
- **Working directory isolation** — Opt in (`isolate: true`, off by default) to copy the eval directory per sample and prevent cross-sample state pollution
- **Setup lifecycle hooks** — Run shell commands before, between (reset), and after samples for fixture management
- **Correctness verification** — Pluggable verifier pipeline: exit code, substring matching, custom scripts, HTTP state checks, LLM judge
- **Multi-runner support** — Built on [agentrunner](https://github.com/driangle/agentrunner) with support for Claude Code, Ollama, Codex, and Aider
- **External eval files** — Reference eval definitions from separate YAML files for reuse across suites
- **Structured reporting** — Markdown and JSON output with per-eval breakdowns, aggregate metrics, and ranked variants
- **Suite validation** — Validate suite YAML structure and required fields without executing

## Quick Start

```bash
# Define your eval suite
cat > suite.yaml <<EOF
version: 1
description: "My first eval suite"
defaults:
  runner: claude-code
evals:
  - id: hello-world
    prompt: "Create a hello world program in Go"
    model: "claude-sonnet-4-6"
    verify:
      - type: output_contains
        values: ["Hello, world!"]
    variants:
      - name: baseline
      - name: with-skill
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
| `--samples N` | Number of runs per variant (default: 1) |
| `--results-dir` | Directory for results output |
| `--variants` | Filter to specific variants |
| `--evals` | Filter to specific eval IDs |
| `--format` | Output format: `markdown`, `json` (default: `markdown`) |
| `-v, --verbose` | Enable debug-level logging |

## Configuration

See the [documentation site](https://driangle.github.io/skival/) for the full configuration schema, verifier reference, and CLI guide.

## Development

### Code size limits

To keep files and functions readable, the following limits are enforced by
`make check-lite`:

| Scope | Limit |
| --- | --- |
| Lines per file (non-test) | 300 |
| Lines per file (`_test.go`) | 500 |
| Lines per function | 40 |
| Statements per function | 25 |

Function-length limits are enforced by [`funlen`](https://golangci-lint.run/usage/linters/#funlen)
via `.golangci.yml` (test files are exempt — table-driven tests run long). File-length
limits are enforced by `scripts/check-file-length.sh` (run as `make lint-filesize`).

If a split is genuinely not worthwhile, add a narrowly-scoped
`//nolint:funlen // <reason>` on the specific function — no blanket directory
exclusions.

### Pre-commit hook

A tracked pre-commit hook (`.githooks/pre-commit`) runs `make check-lite` so
size and lint violations are caught before they land. Install it once per clone:

```bash
make install-hooks
```

This points `core.hooksPath` at the tracked `.githooks/` directory.

## License

MIT
