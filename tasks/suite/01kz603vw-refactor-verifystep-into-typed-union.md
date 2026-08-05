---
id: "01kz603vw"
title: "Refactor VerifyStep god-struct into a typed discriminated union"
status: completed
priority: medium
effort: medium
type: improvement
tags: ["suite", "verify", "yaml-api"]
created: 2026-08-04
verify:
  - type: bash
    run: "go test ./internal/suite/... ./internal/verifier/..."
completed_at: 2026-08-05
---

# Refactor VerifyStep god-struct into a typed discriminated union

## Objective

`VerifyStep` in `internal/suite/suite.go` is a ~20-field struct where a comment
says "Type determines which fields are relevant." Nothing prevents putting a
`url` on a `tcp_check` or a `port` on a `judge` — irrelevant fields are silently
ignored. Ironically the `Probe` types directly above it already use a proper
discriminated union (`HTTP *HTTPProbe`, `File *FileProbe`, ...). `VerifyStep` is
the newer, preferred API but abandoned the safer pattern.

Either adopt per-type payload structs, or (cheaper) add validation that rejects
fields not relevant to the step's `Type`.

## Tasks

- [x] Choose approach: nested typed payloads per verify type, or field-level
      validation keyed on `Type` — chose field-level validation (keeps flat API)
- [x] (If validating) extend `validateVerifySteps` to error when a field not
      belonging to `Type` is set (e.g. `port` on a non-`tcp_check`)
- [ ] (If restructuring) migrate the flat fields into per-type structs and
      update `BuildPipeline` + `migrateCorrectnessToVerify` accordingly — N/A (validation approach)
- [x] Keep the deprecated `correctness`→`verify` migration path working
- [x] Add tests covering wrong-field-for-type rejection

## Acceptance Criteria

- A verify step with a field that does not belong to its `Type` is rejected
  (or made structurally impossible)
- Existing verify examples and the correctness-migration path still work
- `go test ./internal/suite/... ./internal/verifier/...` passes
