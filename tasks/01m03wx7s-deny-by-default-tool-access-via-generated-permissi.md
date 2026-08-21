---
title: "Deny-by-default tool access via generated permission config"
id: "01m03wx7s"
status: completed
priority: high
type: feature
tags: ["tool-access", "runner", "security"]
created: "2026-08-15"
effort: large
completed_at: 2026-08-21
---

# Deny-by-default tool access via generated permission config

## Sub-tasks

This task was split into focused, sequenced slices (see each for detail):

1. **[[01m03awyn-spike-decide-and-verify-claude-code-tool-deny-enfo]]** — Spike:
   decide & verify the enforcement mechanism against a real agent (linchpin; blocks
   the rest).
2. **[[01m039wnp-generate-per-variant-permission-config-and-wire-in]]** — Generate
   the per-variant permission config from `allowed_tools` and wire it into the
   claude-code runner. *(depends on 1)*
3. **[[01m032mr5-default-deny-baseline-and-deprecate-advisory-disal]]** —
   Default-deny baseline (fold in "hermetic mode") + deprecate the advisory
   `--disallowedTools` complement. *(depends on 2)*
4. **[[01m03xmkx-end-to-end-deny-test-docs-and-examples-for-tool-ac]]** — End-to-end
   deny test + docs/examples updates. *(depends on 2, 3)*

## Objective

Make a variant's `allowed_tools` a **structural** whitelist, not an advisory hint.

Today skival passes `allowed_tools` / `disallowed_tools` straight through to the
agent CLI as `--allowedTools` / `--disallowedTools` flags
(`internal/executor/runnercfg.go:35-59` → external `claudecode/runner.go:377-383`).
For built-in tools those flags are **advisory** — the agent can still reach tools
that aren't on the list, and `--disallowedTools` is a hardcoded complement that
inherits a staleness problem: every time the harness adds a new built-in (Skill,
TaskCreate, ToolSearch, …), a hand-maintained deny list silently falls out of date
and the "denied" tool leaks back in.

The robust fix is to generate a **deny-all + allow-listed** permission/settings
config per variant and hand that to the runner, so the whitelist is enforced by
construction rather than by enumerating everything to forbid. This folds the old
"hermetic mode" idea in as the *implementation* of default-deny rather than a
separate opt-in feature — hermeticity becomes the baseline, not a toggle.

**Motivation:** a real run measured what it thought were 4 distinct tool-access
variants but actually measured one configuration, because built-ins leaked past the
advisory flags — ~$75 spent before anyone noticed. See the case study task
[[01m03n2bs-document-tool-access-control-with-the-two-leak-cas]].

## Design notes

- The permission model belongs to the runner CLI (e.g. claude-code settings /
  permission rules with a deny-all default and explicit allow entries), so this is
  primarily a `claude-code` runner concern. Keep it runner-scoped: dispatch already
  happens in `buildRunnerSpecificOpts` (`internal/executor/runnercfg.go:14-32`).
- Do NOT bake in an assumption about which tools exist — generate the config from
  the variant's declared `allowed_tools` plus whatever the suite/CLI provides. Per
  project convention, skival must not assume the user's stack or tool set.
- Per-sample working-directory isolation already exists; write the generated config
  into that sample's working dir / a temp path and point the runner at it.
- Verify the mechanism actually denies unlisted built-ins end-to-end — this is the
  claim the whole feature rests on, so exercise it against a real agent invocation,
  not just a unit test of the config generator.

## Tasks

- [x] Decide the enforcement mechanism for the claude-code runner (generated
      settings/permissions file with deny-all default + allow-listed entries) and
      confirm it actually blocks built-ins, not just MCP/custom tools
- [x] Generate the permission config from a variant's `allowed_tools` at run time
      and pass it to the runner (new option in `buildClaudeCodeOpts`,
      `internal/executor/runnercfg.go`)
- [x] Make default-deny the baseline (fold in "hermetic mode"): with no
      `allowed_tools`, define and document the default posture
- [x] Deprecate reliance on the advisory `--disallowedTools` complement; keep the
      flag path working for runners without a permission-config mechanism
- [x] Add tests: config generation from `allowed_tools`, and an end-to-end check
      that an unlisted built-in is actually denied
- [x] Update `docs/configuration.md` (tool-access section, ~line 441) and the
      `examples/runner-config` + `evals/prompts/tool-access.md` examples

## Acceptance Criteria

- A variant declaring `allowed_tools: [Read, Grep]` cannot use any other tool —
  including newly-added built-ins — without editing the deny list
- Enforcement is generated from the allow list, not a hardcoded complement, so new
  built-ins do not silently leak
- Behaviour is verified against a real agent run, not only unit tests
- Existing suites relying on `allowed_tools` continue to run; the default posture
  when `allowed_tools` is unset is documented
