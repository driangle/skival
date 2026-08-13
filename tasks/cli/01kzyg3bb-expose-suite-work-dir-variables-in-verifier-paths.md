---
title: "Expose suite/work dir variables in verifier paths"
id: "01kzyg3bb"
status: pending
priority: high
type: feature
tags: ["verification", "integrity", "paths"]
created: "2026-08-13"
phase: phase-2
---

# Expose suite/work dir variables in verifier paths

## Objective

Verifiers can't address the suite directory. `check.run` executes in the
working dir, which under `isolate: true` is a temp copy — so a grader living
next to `suite.yaml` is unreachable by relative path.

Today the only workaround is to embed the grader inside the fixture workspace
(e.g. `workspace/.verify/`). That has two problems:

1. It's copied on every sample.
2. **More importantly, the agent under test can read its own grader** — a real
   integrity hole for a benchmarking tool.

The substitution machinery already exists: `${SKIVAL_RUN_DIR}` is substituted
for the exec runner's `events_path`. Extending the same substitution to
verifier paths — exposing `${SKIVAL_SUITE_DIR}` and `${SKIVAL_WORK_DIR}` in
`check`/`command`/`file_contains` paths — would fix this cleanly and let graders
live outside the copied tree.

## Tasks

- [ ] Add `${SKIVAL_SUITE_DIR}` (dir containing `suite.yaml`) and `${SKIVAL_WORK_DIR}` (per-sample working dir) to the path-substitution machinery.
- [ ] Apply substitution to verifier fields: `check.run`/`command` and `file_contains` paths (and any other verifier path inputs).
- [ ] Document that graders should live in the suite dir and be referenced via `${SKIVAL_SUITE_DIR}` so they are not copied into — or readable from — the agent's working tree.
- [ ] Add tests: a grader outside the copied tree, referenced via `${SKIVAL_SUITE_DIR}`, runs correctly and is absent from the sample's isolated workspace.

## Acceptance Criteria

- `${SKIVAL_SUITE_DIR}` and `${SKIVAL_WORK_DIR}` are substituted in verifier `check`/`command`/`file_contains` paths.
- A grader living next to `suite.yaml` is reachable from a verifier without embedding it in the fixture workspace.
- Under `isolate: true`, the grader is not copied into the agent's working tree and is therefore not readable by the agent under test.
- Tests cover both the substitution and the integrity guarantee.
