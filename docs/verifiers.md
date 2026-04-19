# Verifiers

Verifiers check that an AI agent's output is correct. Multiple verifiers can be combined -- all must pass for a sample to be marked as correct.

## Compiles

Runs a user-provided build command in the eval's working directory. Passes if the command exits with code 0.

```yaml
correctness:
  compiles: "go build ./..."
```

The value is any shell command:

```yaml
# Rust
compiles: "cargo check"

# TypeScript
compiles: "npx tsc --noEmit"

# C
compiles: "gcc -o out main.c"
```

## Agent Exits OK

Checks that the agent process exited with code 0.

```yaml
correctness:
  agent_exits_ok: true
```

## Expected Output

Checks that the agent's stdout contains all specified substrings.

```yaml
correctness:
  output:
    contains:
      - "Hello, World!"
      - "success"
```

All substrings must be present. Matching is case-sensitive.

## Script

Runs a custom bash script. Passes if the script exits with code 0.

```yaml
correctness:
  script: "./verify.sh"
```

The script runs in the eval's working directory and has access to any files the agent created.

## State

Makes HTTP requests and checks response bodies. Useful for verifying that a web server or API is in the expected state after the agent runs.

```yaml
correctness:
  state:
    - url: "http://localhost:3000/api/users"
      method: GET
      expect: "alice"
    - url: "http://localhost:3000/api/health"
      method: GET
      expect: "ok"
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `url` | Yes | | URL to request |
| `method` | No | `GET` | HTTP method |
| `expect` | Yes | | Substring expected in the response body |

## Judge

Uses an LLM to evaluate the agent's output against specified criteria. Each criterion is evaluated independently.

```yaml
correctness:
  judge:
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

## Combining Verifiers

Verifiers are evaluated in order: compiles, agent_exits_ok, output, state, script, judge. Evaluation stops at the first failure.

```yaml
correctness:
  agent_exits_ok: true
  output:
    contains:
      - "All tests passed"
  script: "./check-coverage.sh"
  judge:
    - "Code is well-structured and readable"
```

In this example, the agent must exit successfully, print "All tests passed", pass a coverage check script, and satisfy the LLM judge's quality criterion.
