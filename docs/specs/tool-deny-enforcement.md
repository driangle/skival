# Spec: claude-code tool-deny enforcement mechanism

> Decision record for task
> [`01m03awyn`](../../tasks/01m03awyn-spike-decide-and-verify-claude-code-tool-deny-enfo.md)
> (spike). Grounds the implementation sub-task
> [`01m039wnp`](../../tasks/01m039wnp-generate-per-variant-permission-config-and-wire-in.md).

## Problem

A claude-code variant declares `allowed_tools` (e.g. `[Read, Grep]`). We want that
list to behave as a **structural whitelist**: every unlisted tool — including
built-ins like `Bash`, `Write`, `Edit` — is denied, not merely discouraged. Today
`allowed_tools`/`disallowed_tools` map to `--allowedTools`/`--disallowedTools`
(`internal/executor/runnercfg.go:39-44`), and `buildRunOptions` always adds
`--dangerously-skip-permissions` (`internal/executor/singlerun.go:108`). Neither
enforces a whitelist for built-ins.

## Decision

**Use the claude CLI's `--tools` flag as an exclusive built-in whitelist, generated
directly from the variant's `allowed_tools`.**

```
claude --print ... --tools "Read,Grep" -- "<prompt>"
```

`--tools` (CLI help: *"Specify the list of available tools **from the built-in
set**. Use `""` to disable all tools, `"default"` to use all tools, or specify tool
names (e.g. `Bash,Edit,Read`)"*) replaces the default built-in set with exactly the
named tools. Everything unlisted is denied at registration — no complement to
enumerate, and newly-added built-ins are denied automatically because they are not
on the list. This satisfies both the sub-task's "no hardcoded complement" criterion
and the project rule that skival must not assume the user's tool set.

### Upstream change — landed in agentrunner v0.0.2

`v0.0.1` did not emit `--tools`. **agentrunner `v0.0.2` adds `claudecode.WithTools(tools ...string)`**
(→ `--tools "<comma-joined>"`, a single value not repeated flags), alongside
`WithPermissionMode(mode string)` and `WithSettings(fileOrJSON string)`. It also bumps
`MinCLIVersion` to `2.1.0` (the first CLI release that ships `--tools`). The
implementation is therefore just a dependency bump plus wiring:

- **Bump** skival's `go.mod` from `agentrunner/go v0.0.1` to `v0.0.2`.
- skival `internal/executor/runnercfg.go` `buildClaudeCodeOpts`: when
  `allowed_tools` is set, add `claudecode.WithTools(tools...)` (in addition to, or
  instead of, the existing `WithAllowedTools`).
- **CLI-version gate:** `MinCLIVersion` is now `2.1.0`. Any environment running an
  older `claude` will now fail the runner's version check rather than silently
  ignore `--tools` — an intended, safer failure mode, but worth calling out in the
  e2e/docs sub-task (`01m03xmkx`).

### `--dangerously-skip-permissions` may stay

`--tools` operates at tool **registration**, independent of the permission system,
so it is not bypassed by `--dangerously-skip-permissions` (verified: `--disallowedTools`
denial also survives skip-permissions — see Experiment 4). The always-on
`WithSkipPermissions()` at `singlerun.go:108` does **not** need to be removed for
enforcement. (Dropping it for restricted variants is optional hygiene, not a
correctness requirement.)

### Scope: built-ins, not MCP/custom tools

`--tools` restricts the **built-in** set only. MCP and custom tools are governed
separately (via `--mcp-config` / which servers are loaded / `CLAUDE_CONFIG_DIR`).
The leak this task targets is built-ins outliving advisory flags; `--tools` closes
that. A variant that must also deny MCP tools controls them through its MCP config
and `config_dir`. The existing pre-flight leak detector remains the runtime backstop
for anything the flag does not cover.

### Fallback (if `WithTools` cannot land upstream)

`--disallowedTools <every built-in except the allow list>` also denies built-ins
(Experiment 3) and survives skip-permissions (Experiment 4), and needs **no**
upstream change — skival already wires `disallowed_tools`. But it is inferior: there
is no wildcard (Experiment 5), so it requires a **hand-maintained complement** of
built-in tool names, which violates the sub-task's "no hardcoded complement"
criterion and the project's no-assumptions rule. Prefer `--tools`; use disallow only
as a stopgap.

## Empirical evidence

Real `claude` runs, CLI `2.1.220`, `--model haiku`, headless
`--print --output-format stream-json`, in an isolated working dir containing one
readable `notes.txt`. Enforcement was read from the ground truth each run reports:
the `system/init` event's `tools` array, plus whether a tool call returned
`is_error` / produced a side effect.

| # | Flags | Prompt asked for | Result |
|---|-------|------------------|--------|
| 2 | `--allowedTools Read Grep` (no skip-perms) | Bash `echo`, then Read | **Bash executed** (`is_error:false`). `allowed_tools` is **not** exclusive; dropping skip-permissions alone does not deny built-ins. |
| 3 | `--allowedTools Read Grep --disallowedTools Bash` | Bash, then Read | **Bash denied** — absent from `init.tools`; agent: *"I don't have a Bash tool available."* Read worked. |
| 4 | Experiment 3 **+ `--dangerously-skip-permissions`** | Bash, then Read | **Bash still denied** — absent from `init.tools`. Disallow survives skip-permissions. |
| 5 | `--allowedTools Read --disallowedTools '*'` | Read, then Bash | Wildcard denies **nothing** (zero real `tool_use`; `'*'` is a literal name). No wildcard exists → complement must be enumerated. |
| 6 | `--allowedTools Read Grep` + `--disallowedTools <full complement>` | Read, Bash, Write | `init.tools == ["Grep","Read"]`. Bash & Write absent; only Read usable. Complement approach yields a real whitelist. |
| 7 | **`--tools "Read,Grep"`** (no disallow list) | Read, Bash, Write | `init.tools == ["Grep","Read"]`. Bash absent. A hallucinated Write call was **rejected**: `is_error:true`, *"No such tool available: Write. Write is disabled for this session, in subagents as well as here."*; `x.txt` never created. |

Experiment 7 is the acceptance proof: `allowed_tools: [Read, Grep]` expressed as
`--tools "Read,Grep"` blocks unlisted built-ins (including the subagent escape hatch)
with no complement enumeration.

### Notes for the implementer

- **Baseline leak with skip-permissions** (allow-list + `--dangerously-skip-permissions`
  running arbitrary Bash) was not reproduced here because the sandbox's auto-mode
  classifier blocks launching a nested skip-permissions agent that runs arbitrary
  shell. It is documented CLI behavior (`--dangerously-skip-permissions`: *"Bypass
  all permission checks"*) and is corroborated by Experiment 2, where the allow-list
  is already non-exclusive even without skip-permissions.
- **CLI version sensitivity:** resolved in agentrunner v0.0.2 — `MinCLIVersion` is
  now `2.1.0` (the first CLI that ships `--tools`; verified absent in 2.0.0). Runs on
  an older `claude` fail the runner's version check instead of silently dropping the
  flag.
- **Reconcile sub-task framing:** `01m039wnp` currently describes writing a generated
  `.claude/settings.json` into the working dir. A settings-file allow-list is **not**
  exclusive in headless `-p` mode (permissions are advisory there, and skip-perms
  bypasses them). The enforcement lever is the `--tools` **flag**, not a settings
  file — implement config generation as "join `allowed_tools` into a `--tools` value
  via the new `WithTools` option," not as a settings.json writer.
</content>
</invoke>
