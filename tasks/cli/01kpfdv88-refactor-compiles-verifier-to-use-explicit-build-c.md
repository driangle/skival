---
title: "Refactor compiles verifier to use explicit build command"
id: "01kpfdv88"
status: pending
priority: high
type: chore
tags: ["verifier", "breaking-change"]
created: "2026-04-18"
---

# Refactor compiles verifier to use explicit build command

## Objective

Replace auto-detection logic in the `compiles` verifier with a user-provided build command. The `compiles` field should accept a string (the shell command to run) instead of a boolean. This makes the verifier language-agnostic and avoids guessing how to build.

**Before:** `compiles: true` (auto-detects Go/Rust/TypeScript)
**After:** `compiles: "go build ./..."` (user specifies the exact command)

## Tasks

- [ ] Change `Compiles` field in `internal/suite/suite.go` from `*bool` to `string`
- [ ] Update `CompilesVerifier` in `internal/verifier/compiles.go` to run the provided command string instead of auto-detecting. Remove `detectBuildCommand`, `fileExists`, `hasFilesWithExt`
- [ ] Update `BuildPipeline` in `internal/verifier/pipeline.go` to check `c.Compiles != ""` instead of `c.Compiles != nil && *c.Compiles`
- [ ] Update `countVerifiers` in `apps/cli/cmd/validate.go` to check `c.Compiles != ""`
- [ ] Update tests in `internal/verifier/compiles_test.go` — remove detection tests, test with explicit commands
- [ ] Update `docs/verifiers.md` — remove auto-detection table, show string usage
- [ ] Update `docs/examples.md` — change `compiles: true` to `compiles: "go build ./..."`

## Acceptance Criteria

- [ ] `compiles: "go build ./..."` runs `go build ./...` in the eval directory and passes on exit 0
- [ ] `compiles: "gcc -o out main.c"` works for C projects
- [ ] `compiles: true` (boolean) is no longer accepted — suite validation rejects it
- [ ] All existing tests pass, new tests cover the string-based interface
- [ ] Documentation reflects the new string-based syntax
