---
title: "Support multiple skills per treatment (skillsets)"
id: "01kpctpk0"
status: completed
priority: medium
type: feature
tags: ["schema", "skills"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Support multiple skills per treatment (skillsets)

## Objective

Add support for multiple skill files per treatment via a `skills` array field. Currently each treatment only supports a single `skill` string field pointing to one `.md` file. This makes it impossible to compose skillsets (e.g., `[shell-best-practices.md, testing-guidelines.md]`) or compare skillset A vs skillset B as first-class concepts.

The `skills` field should coexist with the existing `skill` field for backward compatibility, with validation ensuring only one is used per treatment.

## Tasks

- [x] Add `Skills []string` field to `Treatment` struct in `internal/suite/suite.go`
- [x] Update `internal/suite/loader.go` to resolve relative paths for all entries in `Skills`
- [x] Add validation in `internal/suite/validate.go`: error if both `skill` and `skills` are set on the same treatment
- [x] Update `internal/executor/executor.go` to read and concatenate all skill files (separated by newlines) into a single `WithAppendSystemPrompt` call
- [x] Update suite loader tests to cover `skills` array loading and path resolution
- [x] Update executor tests to verify multi-skill concatenation
- [x] Update validation tests for mutual exclusivity of `skill` vs `skills`
- [x] Add an example in `examples/` demonstrating skillset comparison

## Acceptance Criteria

- `skills: ["a.md", "b.md"]` in suite.yaml loads and concatenates both files into the appended system prompt
- Relative paths in `skills` are resolved relative to the suite directory
- Setting both `skill` and `skills` on the same treatment produces a validation error
- Existing suites using `skill` (singular) continue to work unchanged
- Skill content is concatenated in array order with `\n\n` separators
- All new code has corresponding tests
