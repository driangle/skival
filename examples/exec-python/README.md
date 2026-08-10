# exec-python

Evaluate an **arbitrary user program** with skival's generic `exec` runner — no
claude-code (or any specific agent) in the loop. Here the program is a tiny
standard-library Python script, but it could be written in any language.

See [docs/exec-runner.md](../../docs/exec-runner.md) for the full contract.

## Files

- `agent.py` — the program under test. It reads the prompt, produces an answer
  on stdout, and (when configured) writes JSONL session events.
- `suite.yaml` — two variants of the same program:
  - **black-box** — prompt delivered on stdin, stdout is the answer.
  - **with-events** — additionally emits JSONL events to
    `${SKIVAL_RUN_DIR}/events.jsonl`, which skival surfaces to the verifier
    pipeline (tool activity, token usage, cost).

## How it works

skival hands the program everything through the environment and stdin:

| Variable             | Meaning                                             |
| -------------------- | --------------------------------------------------- |
| `SKIVAL_PROMPT`      | the eval prompt (also delivered on stdin by default)|
| `SKIVAL_RUN_DIR`     | the working directory for this run                  |
| `SKIVAL_EVENTS_PATH` | where to write JSONL events (only when configured)  |

`agent.py` reverses the text after the first colon, so the prompt
`Reverse this text: hello world` yields `dlrow olleh`, which the
`output_contains` verifier checks.

## Run it

Requires `python3` on your PATH.

```bash
# From the repo root:
skival run examples/exec-python/suite.yaml

# Or just one variant:
skival run examples/exec-python/suite.yaml --variants with-events
```

Both variants should pass. The `with-events` variant additionally writes
`events.jsonl` (git-ignored) into the run directory; its `tool_use`/`tool_result`
events appear in a judge's tool-activity summary, and its `final` event
populates the token/cost columns in the report.
