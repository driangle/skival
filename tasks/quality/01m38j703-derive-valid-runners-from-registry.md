---
id: "01m38j703"
title: "Derive valid runners from the registry"
status: pending
priority: high
effort: small
type: bug
tags: ["suite", "registry", "validation"]
created: 2026-08-12
context: ["internal/suite/validate.go", "internal/registry/registry.go", "apps/cli/cmd/run.go", "README.md"]
verify:
  - type: bash
    run: "go test ./internal/suite/... ./internal/registry/..."
  - type: assert
    check: "A suite naming an unregistered runner fails validation rather than failing at run time"
---

# Derive valid runners from the registry

## Objective

`validRunners` (`internal/suite/validate.go:22`) is a hardcoded map listing
`claude-code`, `ollama`, `codex`, `aider`, `exec`. `defaultRegistry()`
(`apps/cli/cmd/run.go:139`) registers three of them. A `runner: codex` suite
passes `skival validate` cleanly and then fails at run time:

```
$ skival validate codex.yaml && echo OK
OK
$ skival run codex.yaml
t     baseline   1   error   $0.0000  0ms
```

README.md advertises "support for Claude Code, Ollama, Codex, and Aider".
Upstream `agentrunner@v0.0.1` ships only `claudecode` and `ollama`.

The duplicated list also means a user-registered runner would be rejected by
validation, which directly contradicts the harness-agnostic positioning.

## Tasks

- [ ] Pass the registry into suite validation and validate runner names against
      registered factories
- [ ] Remove the hardcoded `validRunners` map
- [ ] Remove Codex and Aider from README and docs, or implement them via the
      `exec` runner as configuration-only adapters (see `01m3je67s`)
- [ ] Reconsider `modelLooksValidForRunner`, which hardcodes `claude-`, `gpt-`,
      `o1-` prefixes against CLAUDE.md's no-assumptions rule
- [ ] Replace the `log.Printf` warning path with `slog`

## Acceptance Criteria

- Validation and execution agree on which runners exist
- Docs list only runners that actually run
