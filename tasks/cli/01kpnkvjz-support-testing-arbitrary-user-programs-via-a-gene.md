---
id: "01kpnkvjz"
title: "Support testing arbitrary user programs via a generic exec runner"
status: pending
priority: medium
effort: large
type: feature
dependencies: []
tags: ["runner", "extensibility"]
created_at: 2026-04-20
---

# Support testing arbitrary user programs via a generic exec runner

## Objective

Today skival can only benchmark built-in runners (claude-code, ollama, codex, aider via agentrunner). Users who write their own agents — e.g. a Python script that calls an arbitrary model/API/orchestrator — have no supported path to evaluate them with skival. Since agents are increasingly bespoke compositions of models, tools, and orchestration code, this is a significant gap.

Add a generic `exec` runner that invokes an arbitrary user-specified command with the eval prompt, captures stdout as the run output, and optionally ingests structured session events from the program so the existing verifier pipeline (judge, tool-activity summarization, probes) continues to work unchanged. The runner must make no assumptions about the user's language, framework, or model — the user describes the invocation in `suite.yaml` (consistent with the project rule that the CLI must not assume a tech stack).

### Example target shape

```yaml
defaults:
  runner: exec

evals:
  - id: summarize
    prompt: "Summarize the attached document in 3 bullets."
    dir: ./my_agent
    variants:
      - name: baseline
        runner_config:
          command: ["python", "agent.py"]
          prompt_via: stdin          # or: env (SKIVAL_PROMPT), arg-file ({prompt_file})
          events_path: "${SKIVAL_RUN_DIR}/events.jsonl"  # optional
    verify:
      - type: output_contains
        any_of: ["•", "-"]
      - type: judge
        criteria: ["Contains exactly three bullets", "Faithful to source"]
```

### Event protocol (opt-in, for rich verification)

If `events_path` (or fd 3 / `$SKIVAL_EVENTS_FD`) is configured, the program may emit JSONL events. Schema is deliberately minimal and shaped to match what `internal/verifier/tool_activity.go` already parses, so no verifier changes are required:

```jsonl
{"type":"tool_use","name":"read_file","input":{"path":"README.md"}}
{"type":"tool_result","tool_use_id":"...","content":"..."}
{"type":"message","role":"assistant","content":"..."}
{"type":"final","text":"...","usage":{"input_tokens":123,"output_tokens":45},"cost_usd":0.012}
```

Programs that emit no events still work — the judge verifier and all output/probe verifiers operate on the final stdout text.

## Tasks

- [ ] Design the exec runner contract and document it in `docs/` (prompt delivery modes, event protocol, env vars injected by skival such as `SKIVAL_RUN_DIR`, `SKIVAL_PROMPT`, `SKIVAL_EVENTS_PATH`)
- [ ] Decide whether the runner lives in the agentrunner dependency or in-tree under `internal/runners/exec/` (likely in-tree, since the protocol is skival-specific)
- [ ] Implement the runner as an `agentrunner.Runner`:
  - [ ] Start: spawn the configured command with cwd = eval `dir`, env merged from suite + skival-injected vars, timeout applied
  - [ ] Deliver the prompt via the configured mode: `stdin` (default), `env`, or `arg-file` with a `{prompt_file}` placeholder
  - [ ] Capture stdout into the final text output; stream stderr to logs
  - [ ] If `events_path` is set, tail the JSONL file and forward events to `session.Messages` as `json.RawMessage`
  - [ ] Parse the terminal `final` event (if any) to populate cost/usage/token fields in `RunResult`
  - [ ] Return exit code; non-zero fails the `agent_exits_ok` verifier as usual
- [ ] Register the runner in `defaultRegistry()` at `apps/cli/cmd/run.go:91`
- [ ] Validate `runner_config` for the exec runner in suite validation (required `command`, enum for `prompt_via`, path checks)
- [ ] Ensure `SummarizeToolActivity` handles the documented event shapes without changes; add fixtures if its assumptions need to widen slightly
- [ ] Add unit tests for the runner (stdin/env/arg-file modes, events file ingestion, timeout, non-zero exit, missing events file is tolerated)
- [ ] Add an `examples/exec-python/` suite with a minimal Python agent (one file, stdlib-only or a trivial `requests` call) plus a README — must be exercised by `TestLoad_Examples`
- [ ] Write `docs/exec-runner.md` covering the contract, event schema, env vars, and two worked examples (black-box and event-emitting)
- [ ] Link the new doc from `docs/configuration.md` and `docs/index.md`

## Acceptance Criteria

- A user can evaluate a Python (or any language) program by setting `runner: exec` and specifying `command` in `runner_config`, with no claude-code dependency in the loop
- Stdout becomes the run's final text and drives `output_contains`, `judge`, and probe verifiers without changes
- Programs that emit JSONL events at `events_path` have those events surfaced to the verifier pipeline such that the judge verifier's tool-activity summary reflects their tool usage
- Exit code is faithfully propagated to the `agent_exits_ok` verifier
- Timeouts, working directory, and env var overrides from the suite/variant are respected identically to other runners
- If `cost_usd` / `usage` are present in the terminal `final` event, they appear in reports; if absent, reports show zero without erroring
- New example suite passes `TestLoad_Examples` and runs end-to-end via `skival run` in the example's README instructions
- Documentation covers the event schema, prompt delivery modes, and injected env vars
- Existing built-in runners and suites continue to work unchanged
