---
id: "01kz63dv0"
title: "Strict YAML decoding to reject unknown/typo'd keys"
status: pending
priority: high
effort: small
type: improvement
tags: ["suite", "yaml-api", "dx"]
created: 2026-08-04
verify:
  - type: bash
    run: "go test ./internal/suite/..."
---

# Strict YAML decoding to reject unknown/typo'd keys

## Objective

`internal/suite/loader.go` parses suites with plain `yaml.Unmarshal`, which
silently ignores unknown keys. This is the root cause that let the docs drift
undetected: a `treatments:` block (or a typo like `varaints:`, `modle:`) is
dropped without warning, and the user only sees a downstream
"at least one variant is required" error — or worse, a silently mis-parsed run.

Switching to a strict decoder turns this whole class of mistakes into a loud,
precise error at load time.

## Tasks

- [ ] Replace `yaml.Unmarshal` in `Load` with a `yaml.Decoder` using
      `dec.KnownFields(true)`
- [ ] Apply the same strict decoding to external eval file refs in
      `resolveFileRefs`
- [ ] Confirm every still-supported deprecated field remains a known struct
      field so strict mode does not break back-compat (`correctness`, `state`,
      `allowed_tools`, `file`)
- [ ] Return a clear error that names the offending key and eval index
- [ ] Add loader tests: unknown top-level key, unknown eval key, unknown
      variant key, and a typo'd `variants` key all error

## Acceptance Criteria

- A suite containing `treatments:` (or any unknown key) fails to load with an
  error naming the unknown field
- All existing `examples/*/suite.yaml` still load and validate
- `go test ./internal/suite/...` passes
