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
- Run each configuration 3 times.
- Check correctness by confirming the agent process exits successfully and by
  making an HTTP request to the running endpoint and asserting the response.

Write only the `suite.yaml` file. Do not create any other files. When you are
finished, the file must be a valid skival suite.
