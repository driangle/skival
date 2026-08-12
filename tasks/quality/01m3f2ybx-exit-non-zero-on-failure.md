---
id: "01m3f2ybx"
title: "Exit non-zero when runs error or verification fails"
status: pending
priority: critical
effort: small
type: bug
tags: ["cli", "ci"]
created: 2026-08-12
context: ["apps/cli/cmd/run.go", "docs/specs/differentiation.md"]
verify:
  - type: bash
    run: "go test ./apps/cli/..."
  - type: assert
    check: "A suite where every run errors exits non-zero"
---

# Exit non-zero when runs error or verification fails

## Objective

`skival run` exits 0 unconditionally. A suite in which every variant fails to
start, every verifier fails, and the runner is unregistered still reports
success to the shell:

```
$ skival run codex.yaml   # 'codex' is not in the registry
t     baseline   1       error   $0.0000  0ms
$ echo $?
0
```

Nothing can be gated on skival in CI, which is the primary deployment target in
the differentiation strategy.

## Tasks

- [ ] Add a `--fail-on` flag with values `never` (default today), `error`,
      `failure` (any failed verification), and pick a sensible default
- [ ] Return a distinct exit code for run errors vs verification failures so CI
      can tell "the harness broke" from "the agent was wrong"
- [ ] Document exit codes in `docs/cli.md`
- [ ] Keep `skival validate` and `skival report` exit semantics unchanged

## Acceptance Criteria

- A suite with an unregistered runner exits non-zero
- A suite where all verifications fail exits non-zero under `--fail-on failure`
- Exit codes are documented and covered by tests
