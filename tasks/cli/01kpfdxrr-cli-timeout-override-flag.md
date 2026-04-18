---
title: "CLI timeout override flag"
id: "01kpfdxrr"
status: completed
priority: low
type: feature
tags: ["cli", "configuration"]
created: "2026-04-18"
completed_at: 2026-04-18
---

# CLI timeout override flag

## Objective

Add a `--timeout` CLI flag to `skival run` that overrides the per-eval timeout defined in suite.yaml. This is useful for quick iteration (lower timeout to fail fast) or giving agents more time on complex evals without editing the YAML.

### Usage

```bash
skival run suite.yaml --timeout 120    # 120 seconds for all evals
```

## Tasks

- [x] Add `--timeout` flag (in seconds) to the `run` command in `cmd/run.go`
- [x] Thread the CLI timeout override through executor options
- [x] Apply CLI timeout as override: CLI flag > eval-level timeout > suite defaults timeout
- [x] Add tests for timeout precedence logic
- [x] Update CLI documentation (cli.md) with the new flag

## Acceptance Criteria

- `--timeout 60` overrides all eval timeouts to 60 seconds
- Omitting `--timeout` uses the existing eval-level or suite default timeout
- A timeout of 0 is rejected with a validation error
- The flag is documented in `skival run --help`
