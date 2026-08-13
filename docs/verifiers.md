# Verifiers

Verifiers check that an AI agent's output is correct. Each eval declares a `verify:` list of steps. Steps run in the order listed, and **all must pass** for a sample to be marked correct; evaluation stops at the first failure.

```yaml
verify:
  - type: agent_exits_ok
  - type: check
    run: "go build ./..."
```

The available step types are `agent_exits_ok`, `check`, `check_output`, `output_contains`, `command`, `file_contains`, `http_check`, `tcp_check`, and `judge`.

## Path variables

The `run` (for `check`, `check_output`, `command`) and `path` (for `file_contains`) fields support two substitution variables, expanded when the pipeline is built:

| Variable | Expands to |
|----------|------------|
| `${SKIVAL_WORK_DIR}` | The per-sample working directory — the isolated temp copy when `isolate: true`, otherwise the eval/variant `dir`. |
| `${SKIVAL_SUITE_DIR}` | The directory containing the loaded `suite.yaml`. |

Any other `${VAR}` falls back to the process environment.

Keep graders next to `suite.yaml` and reference them via `${SKIVAL_SUITE_DIR}` (e.g. `run: "${SKIVAL_SUITE_DIR}/grader.sh"`). Because they live outside the working tree, they are not copied into — nor readable from — the agent's isolated working directory, so the agent under test cannot inspect or tamper with the grader. Use `${SKIVAL_WORK_DIR}` to address files the agent produced in its per-sample working directory.

## `agent_exits_ok`

Checks that the agent process exited with code 0.

```yaml
verify:
  - type: agent_exits_ok
```

## `check`

Runs a shell command in the eval's working directory. Passes if the command exits with code 0. Useful for build/compile checks.

```yaml
verify:
  - type: check
    run: "go build ./..."
```

The `run` value is any shell command:

```yaml
verify:
  - type: check
    run: "cargo check"        # Rust
  - type: check
    run: "npx tsc --noEmit"   # TypeScript
```

## `check_output`

Runs a script with the agent's output piped to its stdin. Passes if the script exits with code 0. The script runs in the eval's working directory and can inspect any files the agent created.

```yaml
verify:
  - type: check_output
    run: "./verify.sh"
```

## `output_contains`

Checks that the agent's output contains all specified substrings. Matching is case-sensitive.

```yaml
verify:
  - type: output_contains
    values:
      - "Hello, World!"
      - "success"
```

## `command`

Runs a shell command and asserts on its exit code and/or stdout.

```yaml
verify:
  - type: command
    run: "go test ./..."
    exits: 0
    stdout_contains: "ok"
```

| Field | Required | Description |
|-------|----------|-------------|
| `run` | Yes | Shell command to run |
| `exits` | No | Expected exit code |
| `stdout_contains` | No | Substring expected in stdout |

## `file_contains`

Checks a file on disk for existence and/or contents. Useful for verifying the agent wrote the expected artifact.

```yaml
verify:
  - type: file_contains
    path: "output.txt"
    exists: true
    contains: "hello world"
```

| Field | Required | Description |
|-------|----------|-------------|
| `path` | Yes | File path (relative to the working directory) |
| `exists` | No | Assert the file exists |
| `contains` | No | Substring expected in the file contents |

## `http_check`

Makes an HTTP request and checks the response. Useful for verifying that a web server or API is in the expected state after the agent runs.

```yaml
verify:
  - type: http_check
    url: "http://localhost:3000/api/users"
    method: GET
    status: 200
    body_contains: "alice"
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `url` | Yes | | URL to request |
| `method` | No | `GET` | HTTP method |
| `status` | No | | Expected HTTP status code |
| `body_contains` | No | | Substring expected in the response body |

## `tcp_check`

Checks that a TCP port is open. Useful for verifying the agent started a server.

```yaml
verify:
  - type: tcp_check
    host: "localhost"
    port: 9090
```

| Field | Required | Description |
|-------|----------|-------------|
| `host` | Yes | Host to connect to |
| `port` | Yes | Port to connect to |

## `judge`

Uses an LLM to evaluate the agent's output against specified criteria. Each criterion is evaluated independently.

```yaml
verify:
  - type: judge
    criteria:
      - "The code handles edge cases (empty input, negative numbers)"
      - "The implementation uses idiomatic Go patterns"
      - "Error messages are user-friendly"
```

The judge receives the original prompt, the agent's output, and each criterion, then returns a pass/fail verdict.

By default the judge uses `claude-haiku-4-5-20251001`. Override it with `model` on the judge step:

```yaml
verify:
  - type: judge
    criteria: ["The code handles edge cases"]
    model: "claude-sonnet-4-6"
```

`defaults.judge_model` applies to any judge step that doesn't set its own `model` (see [Configuration](configuration.md)).

The `judge` step grades each run in isolation (pass/fail). To instead compare the outputs of the variants that *passed* an eval and rank them by relative quality, see [Comparative Judging](configuration.md#comparative-judging).

## Combining Verifiers

List multiple steps under `verify:`. They run in order and evaluation stops at the first failure.

```yaml
verify:
  - type: agent_exits_ok
  - type: output_contains
    values:
      - "All tests passed"
  - type: check_output
    run: "./check-coverage.sh"
  - type: judge
    criteria:
      - "Code is well-structured and readable"
```

In this example, the agent must exit successfully, print "All tests passed", pass a coverage check script, and satisfy the LLM judge's quality criterion.
