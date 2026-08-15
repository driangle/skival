---
title: "tool_not_used verifier as a leakage backstop"
id: "01m037pwn"
status: pending
priority: low
type: feature
tags: ["tool-access", "verifier"]
created: "2026-08-15"
dependencies: ["01m03wx7s", "01m030n6n", "01m036fad"]
effort: medium
---

# tool_not_used verifier as a leakage backstop

## Objective

Add a `tool_not_used` verifier so a suite can **fail** (not just warn) when a sample
touches a forbidden tool.

This is intentionally the **backstop**, demoted below the other three tool-access
tasks. Once deny-by-default enforcement
([[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]]), the pre-flight
warning ([[01m030n6n-pre-flight-tool-leak-detection-from-system-init-ev]]), and the
report census ([[01m036fad-per-variant-tool-census-in-the-report]]) make leakage
rare and visible, some suites will still want a hard assertion that turns leakage
into a failed run — e.g. proving a "no tools" variant genuinely used no tools. That's
what this verifier provides.

## Design notes

- Verifiers register in two places, both required (per code map):
  - Runtime dispatch: add a `case` in `buildStepVerifier`
    (`internal/verifier/pipeline.go:103`) returning a `namedVerifier`.
  - Schema/validation: add the type to `validVerifyTypes` and `verifyTypeFields`
    (`internal/suite/verify.go:5,20`), a branch in `validateVerifyStepRequired`
    (line 94), and a field on `VerifyStep` (`internal/suite/suite.go:90-122`) for the
    forbidden tool-name list.
- The verifier interface already carries what's needed: `VerifyInput.Conversation`
  (`internal/verifier/verifier.go:9-18`) holds the raw tool_use blocks, so the
  verifier can inspect tool activity with no new plumbing. Model the traversal on
  `internal/verifier/tool_activity.go` (or reuse the shared tool-count field added by
  the census task, if that lands first).
- Config shape (illustrative): `type: tool_not_used` with `tools: [Skill, TaskCreate]`
  — fail if any listed tool was invoked. Consider a complementary `only_tools`
  (fail if any tool *outside* the set was used) if cheap, but keep this task scoped to
  the deny direction.
- Don't assume a fixed tool taxonomy; match on the tool names the suite specifies.

## Tasks

- [ ] Add a `VerifyStep` field for the forbidden tool list and register
      `tool_not_used` in `internal/suite/verify.go` (all three maps/branches)
- [ ] Implement the verifier (new file in `internal/verifier/`, modeled on
      `tool_activity.go`) and dispatch it in `pipeline.go`'s `buildStepVerifier`
- [ ] Tests: pass when no forbidden tool used, fail when one is, across both
      conversation shapes
- [ ] Document the verifier in `docs/verifiers.md`

## Acceptance Criteria

- A suite can declare `type: tool_not_used` with a tool list and the run fails when
  the agent invokes any of those tools
- The verifier reads the existing conversation data — no new runner plumbing
- Passes cleanly when the forbidden tools are absent
- Documented in `docs/verifiers.md`
