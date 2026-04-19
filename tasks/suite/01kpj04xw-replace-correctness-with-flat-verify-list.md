---
id: "01kpj04xw"
title: "Replace correctness with flat verify list"
status: completed
priority: high
dependencies: ["01kpgv3xy"]
tags: ["correctness", "yaml-api"]
created_at: 2026-04-19
completed_at: 2026-04-19
---

# Replace correctness with flat verify list

## Objective

Replace the `correctness` object (mix of shortcut fields + `probes` list) with a single `verify` list where every check is a typed step. This unifies all verification into one shape and one mental model.

### Proposed YAML shape

```yaml
verify:
  - type: agent_exits_ok

  - type: check
    run: "go build ./..."

  - type: check_output
    run: "./verify.sh"

  - type: output_contains
    values: ["PASS", "all tests green"]

  - type: command
    name: tests_pass
    run: pytest tests/ -q

  - type: file_contains
    path: output.txt
    contains: "hello"

  - type: http_check
    url: http://localhost:3000/health
    status: 200

  - type: tcp_check
    host: localhost
    port: 5432

  - type: judge
    criteria:
      - "Does it explain X?"
```

## Tasks

- [x] Define `VerifyStep` struct with `Type` discriminator and per-type fields in `internal/suite/suite.go`
- [x] Replace `Correctness` field on `Eval` with `Verify []VerifyStep`
- [x] Update `BuildPipeline` to construct verifiers from `[]VerifyStep` instead of `Correctness`
- [x] Add YAML validation for `verify` steps (valid type, required fields per type)
- [x] Migrate `correctness` → `verify` in loader with deprecation warning (convert all old fields to typed steps)
- [x] Update all existing verifiers to work with the new step types
- [x] Update `examples/correctness/suite.yaml` to use `verify`
- [x] Update all existing tests (pipeline, loader, validation) for the new structure
- [x] Add tests for the `correctness` → `verify` migration path
- [x] Ensure `TestLoad_Examples` passes

## Acceptance Criteria

- `verify` is a flat list of typed steps — one shape for all checks
- All existing verification types work: `agent_exits_ok`, `check`, `check_output`, `output_contains`, `command`, `file_contains`, `http_check`, `tcp_check`, `judge`
- Old `correctness` suites still load via migration with deprecation warning
- Pipeline order follows list order (steps run in the order they appear)
- `TestLoad_Examples` passes
- Each step supports an optional `name` field for reporting
