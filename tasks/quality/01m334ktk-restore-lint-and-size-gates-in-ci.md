---
id: "01m334ktk"
title: "Restore lint and file-size gates in CI"
status: pending
priority: critical
effort: small
type: chore
tags: ["ci", "quality"]
created: 2026-08-12
context: [".golangci.yml", ".github/workflows/ci.yaml", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "golangci-lint run ./..."
  - type: bash
    run: "make lint-filesize"
---

# Restore lint and file-size gates in CI

## Objective

The `lint` job has failed on every CI run since 2026-04-19. `.golangci.yml` is
in v1 format and current golangci-lint (2.x) rejects it outright:

```
Error: can't load config: unsupported version of the configuration: ""
```

Because `make lint-filesize` is a later step in the same job, GitHub marks it
`skipped`. The file-size and function-length limits that README.md and CLAUDE.md
both describe as *enforced* have therefore never run in CI, and the tracked
pre-commit hook (`make check-lite`) fails for anyone who installs it.

Separately, `golangci-lint` reports 28 real `errcheck` violations — default
linters, so these were failing on content as well as on config.

## Tasks

- [ ] Migrate `.golangci.yml` to the v2 schema (`version: "2"`, `linters.settings`,
      `linters.exclusions.rules`)
- [ ] Pin the golangci-lint version in `.github/workflows/ci.yaml` so a major
      release cannot silently break the gate again
- [ ] Move `make lint-filesize` into its own CI job so a lint failure can never
      skip the size check
- [ ] Fix the 28 `errcheck` violations (unchecked `Close()` in
      `internal/runners/exec/{events,process}.go`,
      `internal/verifier/{http_probe,state,tcp_probe}.go` and tests)
- [ ] Confirm `make check` passes end to end on a clean clone

## Acceptance Criteria

- `golangci-lint run ./...` exits 0 with a current 2.x binary
- CI is green on `main`
- A deliberate 301-line non-test Go file fails CI even when lint passes
