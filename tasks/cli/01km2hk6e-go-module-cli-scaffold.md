---
title: "Go module init and CLI scaffold"
id: "01km2hk6e"
status: completed
priority: high
type: feature
tags: ["phase-1", "cli"]
created: "2026-03-19"
phase: phase-1
---

# Go module init and CLI scaffold

## Objective

Initialize the Go module and monorepo structure with a cobra-based CLI. Wire up `run` and `report` subcommands as stubs with all planned flag definitions.

## Tasks

- [x] Run `go mod init` for the skival module
- [x] Create `apps/cli/main.go` with cobra root command
- [x] Create `apps/cli/cmd/run.go` with `skival run <suite.yaml>` stub and flags: `--samples`, `--results-dir`, `--treatments`, `--evals`, `--model`, `--format`
- [x] Create `apps/cli/cmd/report.go` with `skival report <results-dir>` stub and flags: `--format`
- [x] Verify the CLI builds and `--help` output is correct

## Acceptance Criteria

- `go build ./apps/cli` succeeds
- `skival --help` shows root help with `run` and `report` subcommands
- `skival run --help` shows all defined flags
- `skival report --help` shows its flags
