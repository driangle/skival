You are configuring a skival eval suite. skival is a CLI that evaluates and
compares AI agent configurations by measuring correctness, cost, and speed. It
can restrict which tools an agent is allowed to use and measure the effect.

Create a skival suite file named `suite.yaml` in the current directory that
compares tool-access levels on a single task:

- The task: "Add a new REST endpoint `GET /health` that returns HTTP 200 with a
  JSON body `{\"status\":\"ok\"}`."
- Compare three configurations of the same model (`claude-sonnet-4-6`):
  one with full tool access, one restricted to read-only tools, and one that is
  not allowed to use the shell.
  - For claude-code, `allowed_tools` is an **exclusive deny-by-default whitelist**:
    any built-in not listed is denied. Express *full tool access* as
    `allowed_tools: [default]` (omitting it denies every built-in), the *read-only*
    config as `allowed_tools: [Read, Grep, Glob]`, and *no shell* as an allow list
    that simply omits `Bash`.
- Run each configuration 3 times.
- Check correctness by confirming the agent process exits successfully and by
  making an HTTP request to the running endpoint and asserting the response.

Reference: skival's tool-access model (deny-by-default enforcement, the pre-flight
leak warning, the per-variant tool census, and the `tool_not_used` backstop verifier)
is documented under "Tool Access Control" in `docs/configuration.md`.

Write only the `suite.yaml` file. Do not create any other files. When you are
finished, the file must be a valid skival suite.
