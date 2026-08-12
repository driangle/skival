---
id: "01m39zy40"
title: "GitHub Action, release polish, and honest positioning docs"
status: pending
priority: medium
effort: medium
type: docs
tags: ["distribution", "ci", "docs"]
created: 2026-08-12
dependencies: ["01m3jbzwq"]
context: [".github/workflows/release.yml", "README.md", "docs/index.md", "docs/specs/differentiation.md"]
verify:
  - type: assert
    check: "A PR can be gated on skival with a single GitHub Action step and no runtime installed"
---

# GitHub Action, release polish, and honest positioning docs

## Objective

The single static binary is skival's cheapest adoption advantage, and it is
currently unexploited: there is no published action, and the Homebrew tap task
(`01kpehm8g`) is still open.

The positioning also needs to be stated plainly. Naming the cases where a
competitor is the better answer is cheap, verifiable, and buys more credibility
than a feature matrix — particularly against promptfoo, which genuinely is the
better choice for many users.

## Tasks

- [ ] Publish a composite GitHub Action wrapping `skival run --fail-on regression`
- [ ] Finish the Homebrew tap (`01kpehm8g`) and verify the release workflow
      produces static binaries for linux/darwin on amd64/arm64
- [ ] Add `skival init` to scaffold a starter suite for an existing project
- [ ] Rewrite the README opening around the verdict story rather than the
      feature list
- [ ] Add a docs page: "when to use promptfoo instead" — breadth of providers
      and assertions, red teaming, and the hosted viewer are all cases where the
      answer is honestly "not skival"
- [ ] Update `docs/PLAN.md`, which describes an architecture the code no longer
      has (`internal/runner/`, a sandboxing executor, `verifier/script.go`)

## Acceptance Criteria

- A PR can be gated on skival in one workflow step with no runtime installed
- `brew install` works
- The docs state, without hedging, when a competitor is the better tool
