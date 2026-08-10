# Exec Runner

The `exec` runner evaluates **any program you can invoke from a command line** —
a Python script, a compiled binary, a shell pipeline, a bespoke agent that calls
whatever model, tools, and orchestration you like. skival makes no assumptions
about your language, framework, or model; you describe the invocation in
`runner_config`.

skival runs your command, feeds it the eval prompt, captures **stdout as the run
output**, and (optionally) ingests a stream of **JSONL session events** so the
judge, tool-activity summary, and cost/usage reporting work exactly as they do
for the built-in runners.

See the runnable [`examples/exec-python`](https://github.com/driangle/skival/tree/main/examples/exec-python)
suite for a complete, working setup.

## Quick start

```yaml
version: 1
defaults:
  runner: exec

evals:
  - id: summarize
    prompt: "Summarize the attached document in 3 bullets."
    dir: ./my_agent          # command runs with this as its working directory
    verify:
      - type: output_contains
        values: ["- "]
    variants:
      - name: baseline
        runner_config:
          command: ["python3", "agent.py"]
          prompt_via: stdin
```

Unlike the model-backed runners, an exec variant needs **no `model`** — the
invocation contract lives entirely in `runner_config`.

## `runner_config` reference

| Key           | Required | Description                                                                 |
| ------------- | -------- | --------------------------------------------------------------------------- |
| `command`     | **Yes**  | Program and arguments as a list, e.g. `["python3", "agent.py"]`.            |
| `prompt_via`  | No       | How the prompt is delivered: `stdin` (default), `env`, or `arg-file`.       |
| `prompt_env`  | No       | Env var name used in `env` mode (default `SKIVAL_PROMPT`).                  |
| `events_path` | No       | Path skival reads JSONL events from. Supports `${SKIVAL_RUN_DIR}` and other `${VAR}` expansion; a relative path is anchored to the working directory. |

`runner_config` deep-merges across `defaults` → eval → variant, like every other
runner.

## Prompt delivery modes

| Mode       | How the prompt reaches your program                                                                 |
| ---------- | -------------------------------------------------------------------------------------------------- |
| `stdin`    | Written to the program's standard input (default).                                                 |
| `env`      | Passed in an environment variable — `SKIVAL_PROMPT`, or the name you set via `prompt_env`.          |
| `arg-file` | Written to a temp file; the `{prompt_file}` token in `command` is replaced with that file's path.   |

`arg-file` example:

```yaml
runner_config:
  command: ["./my-agent", "--prompt-file", "{prompt_file}"]
  prompt_via: arg-file
```

Regardless of mode, `SKIVAL_PROMPT` is **always** present in the environment, so
a program can read the prompt from there if that is convenient.

## Injected environment variables

skival injects these into your program's environment (on top of the parent
environment and any `env:` overrides from the suite/variant):

| Variable             | Always set? | Meaning                                                       |
| -------------------- | ----------- | ------------------------------------------------------------- |
| `SKIVAL_PROMPT`      | Yes         | The eval prompt.                                              |
| `SKIVAL_RUN_DIR`     | When a working dir is set | The working directory for this run (isolated copy when `isolate: true`). |
| `SKIVAL_EVENTS_PATH` | When `events_path` is set | The resolved path your program should write JSONL events to. |

Working directory, per-variant `env`, and timeouts behave identically to the
other runners.

## Exit code and correctness

A **non-zero exit is a normal outcome**, not a runner failure: skival captures
the exit code and hands it to the `agent_exits_ok` verifier, which fails the run
when the code is non-zero. This lets you assert success/failure explicitly:

```yaml
verify:
  - type: agent_exits_ok        # fails if your program exits non-zero
  - type: output_contains
    values: ["expected text"]
```

Programs that emit **no** events still work fully — stdout drives
`output_contains`, `judge`, and the probe verifiers.

## Event protocol (opt-in)

When `events_path` is configured, your program may write **JSONL** (one JSON
object per line) describing its session. skival reads the file **after your
program exits** and forwards each event into the verifier pipeline. A missing or
unreadable file is tolerated — you are never required to produce events.

The schema is deliberately minimal and matches what skival's tool-activity
summarizer already understands:

```jsonl
{"type":"tool_use","name":"read_file","input":{"path":"README.md"}}
{"type":"tool_result","tool_use_id":"1","content":"…file contents…"}
{"type":"message","role":"assistant","content":"…"}
{"type":"final","text":"…","usage":{"input_tokens":123,"output_tokens":45},"cost_usd":0.012}
```

| Event type    | Purpose                                                                                  |
| ------------- | ---------------------------------------------------------------------------------------- |
| `tool_use`    | Records a tool call (`name`, `input`). Surfaced in the judge's tool-activity summary.    |
| `tool_result` | Records the result of a tool call (`content`). Surfaced in the tool-activity summary.    |
| `message`     | A free-form assistant/user message. Carried through but not treated as tool activity.    |
| `final`       | Terminal event carrying `usage` and `cost_usd` (and an optional `text` fallback).        |

### The `final` event

- `usage.input_tokens`, `usage.output_tokens`,
  `usage.cache_creation_input_tokens`, and `usage.cache_read_input_tokens`
  populate the run's token counts.
- `cost_usd` populates the run's cost, which flows into the cost column and
  ranking. If absent, cost is reported as `0` without error.
- `text` is used as the run output **only when the program wrote nothing to
  stdout** — otherwise stdout wins.

## Two worked examples

### Black box (no events)

The simplest possible integration: the program reads the prompt and prints an
answer. Correctness is judged purely from stdout.

```yaml
variants:
  - name: black-box
    runner_config:
      command: ["python3", "agent.py"]
      prompt_via: stdin
verify:
  - type: output_contains
    values: ["dlrow olleh"]
  - type: agent_exits_ok
```

```python
# agent.py
import os, sys
prompt = sys.stdin.read() or os.environ["SKIVAL_PROMPT"]
text = prompt.split(":", 1)[1].strip()
sys.stdout.write(text[::-1])
```

### Event-emitting (rich verification)

The same program additionally writes JSONL events so a judge can see the tools
it used and reports can show token usage and cost.

```yaml
variants:
  - name: with-events
    runner_config:
      command: ["python3", "agent.py"]
      prompt_via: stdin
      events_path: "${SKIVAL_RUN_DIR}/events.jsonl"
verify:
  - type: judge
    criteria: ["The answer is the input reversed"]
  - type: agent_exits_ok
```

```python
# agent.py (events portion)
import json, os
path = os.environ.get("SKIVAL_EVENTS_PATH")
if path:
    with open(path, "w") as f:
        f.write(json.dumps({"type": "tool_use", "name": "reverse",
                            "input": {"text": prompt}}) + "\n")
        f.write(json.dumps({"type": "final", "text": answer,
                            "usage": {"input_tokens": len(prompt),
                                      "output_tokens": len(answer)},
                            "cost_usd": 0.0}) + "\n")
```
