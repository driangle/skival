#!/usr/bin/env python3
"""A minimal skival exec-runner agent (standard library only).

It reads the eval prompt, "reverses" the text after the first colon, writes the
answer to stdout, and — when skival advertises an events file — emits a small
JSONL session-event stream so the judge/tool-activity pipeline has something to
inspect.

skival hands the agent everything it needs through the environment and stdin:

  SKIVAL_PROMPT       the eval prompt (also delivered on stdin by default)
  SKIVAL_RUN_DIR      the working directory for this run
  SKIVAL_EVENTS_PATH  where to write JSONL events (only set when configured)
"""

import json
import os
import sys


def read_prompt() -> str:
    """Return the prompt from stdin, falling back to SKIVAL_PROMPT."""
    data = sys.stdin.read()
    if data.strip():
        return data
    return os.environ.get("SKIVAL_PROMPT", "")


def answer_for(prompt: str) -> str:
    """The agent's 'reasoning': reverse the text after the first colon."""
    text = prompt.split(":", 1)[1].strip() if ":" in prompt else prompt.strip()
    return text[::-1]


def emit_events(path: str, prompt: str, answer: str) -> None:
    """Write a minimal JSONL event stream skival can surface to the verifier."""
    events = [
        {"type": "tool_use", "name": "reverse", "input": {"text": prompt}},
        {"type": "tool_result", "tool_use_id": "1", "content": answer},
        {
            "type": "final",
            "text": answer,
            "usage": {"input_tokens": len(prompt), "output_tokens": len(answer)},
            "cost_usd": 0.0,
        },
    ]
    with open(path, "w", encoding="utf-8") as f:
        for event in events:
            f.write(json.dumps(event) + "\n")


def main() -> None:
    prompt = read_prompt()
    answer = answer_for(prompt)

    events_path = os.environ.get("SKIVAL_EVENTS_PATH")
    if events_path:
        emit_events(events_path, prompt, answer)

    sys.stdout.write(answer)


if __name__ == "__main__":
    main()
