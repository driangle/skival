---
id: "01m03n2bs"
title: "Document tool-access control with the two-leak case study"
status: pending
priority: medium
type: docs
effort: small
dependencies: ["01m036fad", "01m030n6n"]
tags: ["tool-access", "docs"]
created: 2026-08-15
---

# Document tool-access control with the two-leak case study

## Objective

Document how tool-access control works in skival, anchored on a concrete case study:
a real 4-variant suite that silently measured **one** configuration because tools
leaked past the advisory `allowed_tools` flags — and both leaks were invisible in
skival's own report, surfacing only after ~$75 and hand-grepping JSONL.

The point of the case study is the deeper lesson, not just the fix: **a benchmark
that can't show you what the agent had access to can't be trusted, however correct
its enforcement is.** The docs should make that the organizing principle for the
tool-access section.

## Design notes

- Depends on the observability features landing so the docs can point at real output:
  the pre-flight warning ([[01m030n6n-pre-flight-tool-leak-detection-from-system-init-ev]])
  and the report tool census ([[01m036fad-per-variant-tool-census-in-the-report]]).
- Cross-reference the enforcement model
  ([[01m03wx7s-deny-by-default-tool-access-via-generated-permissi]]) and the backstop
  verifier ([[01m037pwn-tool-not-used-verifier-as-a-leakage-backstop]]).
- Homes: extend the tool-access section of `docs/configuration.md` (~line 441) and
  `docs/verifiers.md`; the `evals/prompts/tool-access.md` and
  `examples/runner-config` examples can link to it.
- Mention that skival (or the client) can use **vibeview** to inspect a session and
  see exactly which tools were available and used — a manual counterpart to the
  built-in census.

## Tasks

- [ ] Write the case study: what was configured (4 variants), what actually ran (1),
      why the advisory flags didn't enforce, and why it was invisible in the report
- [ ] Document the enforcement model (deny-by-default / generated permission config)
      and when to rely on it
- [ ] Document the observability tools: pre-flight leak warning + per-variant tool
      census, with example output; note vibeview for manual session inspection
- [ ] Document the `tool_not_used` backstop verifier and when to use warn vs. fail
- [ ] Link the new content from `docs/configuration.md`, `docs/verifiers.md`, and the
      tool-access example(s)

## Acceptance Criteria

- A reader can understand how tool access is enforced, how to confirm what tools a
  variant actually had and used, and how to make leakage fail a run
- The case study concretely explains the two leaks and why they were invisible
- The "you can't trust a benchmark that can't show tool access" lesson is stated
  explicitly and ties the enforcement + observability features together
- Docs cross-reference each other and the relevant examples
