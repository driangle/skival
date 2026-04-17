---
title: "Add runner and runner_config fields to suite types"
id: "01kpcvhw5"
status: completed
priority: critical
type: feature
effort: small
tags: ["backend", "suite"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Add runner and runner_config fields to suite types

## Objective

Add `Runner` and `RunnerConfig` fields to the `Defaults`, `Eval`, and `Treatment` structs in the suite package. This is the foundational type change that all other multi-runner work depends on.

## Tasks

- [x] Add `Runner string` and `RunnerConfig map[string]any` fields (with yaml tags) to `Defaults`, `Eval`, and `Treatment` structs in `internal/suite/suite.go`
- [x] Remove `AllowedTools []string` from `Treatment` struct (it moves into `runner_config`)
- [x] Add backward-compat shim in the loader: if `AllowedTools` is set during loading, migrate it to `RunnerConfig["allowed_tools"]` with a deprecation log warning
- [x] Update any existing tests that reference `AllowedTools` on Treatment

## Acceptance Criteria

- [x] Suite YAML with `runner` and `runner_config` fields parses correctly into the new struct fields
- [x] Existing suites using `allowed_tools` at the treatment level still load correctly (migrated into `runner_config.allowed_tools`)
- [x] A deprecation warning is logged when `allowed_tools` migration occurs
- [x] All existing tests pass
- [x] New unit tests cover the `AllowedTools` → `runner_config` migration
