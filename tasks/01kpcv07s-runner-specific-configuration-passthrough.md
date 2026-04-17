---
title: "Runner-specific configuration passthrough"
id: "01kpcv07s"
status: pending
priority: high
type: feature
tags: ["schema", "runner"]
created: "2026-04-17"
---

# Runner-specific configuration passthrough

## Objective

Add a generic `runner_config` map to treatments that passes runner-specific options through to the selected runner. Currently `allowed_tools` is hardcoded to `claudecode.WithAllowedTools` — other runners have different configuration knobs (e.g., Codex sandbox settings, Aider edit format, approval modes).

This depends on multi-runner support being implemented first.

## Tasks

- [ ] Add `RunnerConfig map[string]any` field to `Treatment` struct in `internal/suite/suite.go`
- [ ] Update suite loader to parse `runner_config` as a free-form YAML map
- [ ] Define a `RunnerConfigurer` interface or extend the runner factory to accept config maps
- [ ] Move `allowed_tools` handling from executor into Claude Code runner's config handler
- [ ] Implement config passthrough in executor — runner receives its config map and applies it
- [ ] Add validation: warn on unrecognized keys for known runners
- [ ] Add loader tests for `runner_config` parsing
- [ ] Add executor tests for config passthrough

## Acceptance Criteria

- `runner_config` in suite.yaml is parsed and passed to the selected runner
- Claude Code runner interprets `allowed_tools` from `runner_config` (backward compat: top-level `allowed_tools` still works)
- Unknown config keys produce a warning, not an error (runners may support keys skival doesn't know about)
- Each runner type can define its own config schema/handling
- Existing suites using `allowed_tools` at the treatment level continue to work
