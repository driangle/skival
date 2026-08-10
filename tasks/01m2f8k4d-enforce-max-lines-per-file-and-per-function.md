---
id: "01m2f8k4d"
title: "Enforce max lines per file and per function in check-lite + pre-commit"
status: pending
priority: medium
effort: medium
type: chore
tags: ["tooling", "lint", "tech-debt"]
created: 2026-08-10
---

# Enforce max lines per file and per function in check-lite + pre-commit

## Objective

There is currently no size limit on Go files or functions. The repo has no
`.golangci.yml` at all, so `make lint` runs golangci-lint's default linter set,
which enforces neither. Nothing stops a file from growing into a "god file"—
`internal/suite/loader_test.go` is already 2010 lines and
`internal/executor/executor.go` is 688.

Add enforced limits, wire them into `make check-lite`, and install a pre-commit
hook that runs `check-lite` so violations are caught before they land.

## Proposed limits

Starting points — adjust once the violation count from the initial run is known:

| Scope | Limit |
| --- | --- |
| Lines per file (non-test) | 300 |
| Lines per file (`_test.go`) | 500 |
| Lines per function | 40 |
| Statements per function | 25 |

These are strict on purpose: they should force splits, not merely catch runaway
files. Expect a meaningful refactor pass — several existing files are 2–6x over.

## Tasks

- [ ] Add a `.golangci.yml` that keeps the current default linters and enables
      function-length enforcement (`funlen`) plus file-length enforcement
      (`revive`'s `file-length-limit` rule, since golangci-lint has no standalone
      file-length linter)
- [ ] Confirm the pinned golangci-lint version supports `file-length-limit`; if
      not, either bump it or implement the file check as a small script invoked
      from a `make lint-filesize` target
- [ ] Set per-path overrides so `_test.go` files get the larger file budget
      (table-driven tests legitimately run long) while still being bounded
- [ ] Run the linters and record the current violation list. File-length
      violations as of 2026-08-10 (9 files; re-measure at implementation time,
      the tree was dirty):

      ```
      2010  internal/suite/loader_test.go
      1663  internal/executor/executor_test.go
       766  internal/suite/validate_test.go
       688  internal/executor/executor.go
       519  internal/report/rank_test.go
       414  internal/suite/validate.go
       407  internal/report/html.go
       387  internal/suite/loader.go
       343  internal/report/markdown.go
      ```

      Function-length violations are unknown until `funlen` runs.
- [ ] Split or refactor existing violations, or add narrowly-scoped `//nolint`
      exemptions with a reason where a split is not worth it — no blanket
      directory exclusions
- [ ] Ensure `make check-lite` fails on a violation (it already depends on
      `lint`; add any new target to the `check-lite` prerequisite list)
- [ ] Add a checked-in pre-commit hook that runs `make check-lite`, plus a
      `make install-hooks` target (or `core.hooksPath` pointing at a tracked
      `.githooks/` directory) so the hook is set up with one command —
      `.git/hooks/` is not version-controlled and is currently empty
- [ ] Document the limits and hook installation in `README.md` / `CLAUDE.md`
- [ ] Add a test covering the new tooling: a check that fails if a file or
      function exceeds the limits, or a script test if the file check is
      implemented as a script

## Acceptance Criteria

- A Go file exceeding the file-line limit fails `make lint` and therefore
  `make check-lite`
- A function exceeding the line/statement limit fails `make lint`
- `make check-lite` passes on a clean tree (all pre-existing violations are
  resolved or explicitly exempted with a documented reason)
- A fresh clone can install the pre-commit hook with a single documented command
- The hook blocks a commit that introduces a violation
- Limits are documented and discoverable

## Notes

- Keep the config declarative in `.golangci.yml` where possible; only fall back
  to a custom script if the linter cannot express the file-length rule.
- The hook should be fast enough to not be annoying — `check-lite` already skips
  tests, which is why it is the right target.
