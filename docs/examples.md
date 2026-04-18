# Examples

The [examples/](https://github.com/driangle/skival/tree/main/examples) directory contains self-contained suites that demonstrate each skival feature. Clone the repo and run any example directly:

```bash
skival run examples/minimal/suite.yaml
```

## Minimal

The simplest valid suite — one eval, two treatments.

```yaml
version: 1

evals:
  - id: hello
    prompt: "Write a file called hello.txt containing 'Hello, World!'"
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
      variations:
        - name: opus-model
          model: "claude-opus-4-6"
```

[View source](https://github.com/driangle/skival/tree/main/examples/minimal)

## Defaults

Suite-level defaults inherited by all evals. Individual evals can override any default.

```yaml
version: 1

defaults:
  model: "claude-sonnet-4-6"
  samples: 5
  timeout: 120
  runner: claude-code
  runner_config:
    max_turns: 10

evals:
  - id: uses-defaults
    prompt: "Create a file called greeting.txt with a friendly greeting."
    treatments:
      control:
        name: baseline

  - id: overrides-defaults
    prompt: "Write a script that prints the current date."
    samples: 3        # overrides suite default
    timeout: 60       # overrides suite default
    model: "claude-opus-4-6"
    treatments:
      control:
        name: baseline
```

[View source](https://github.com/driangle/skival/tree/main/examples/defaults)

## File References

Split eval definitions into separate files to keep large suites organized.

```yaml
version: 1

defaults:
  model: "claude-sonnet-4-6"
  samples: 2

evals:
  - file: evals/string-reverse.yaml
  - file: evals/fibonacci.yaml
```

[View source](https://github.com/driangle/skival/tree/main/examples/file-refs)

## Correctness Verification

Every verification mode in one suite: `compiles`, `agent_exits_ok`, `output`, `script`, `state`, and `judge`.

```yaml
evals:
  # Verify compilation
  - id: compiles-check
    prompt: "Write a Go program in main.go that prints 'hello'."
    correctness:
      compiles: "go build ./..."

  # Verify the agent exits successfully
  - id: agent-exits-ok-check
    prompt: "Write a shell script run.sh that exits with code 0."
    correctness:
      agent_exits_ok: true

  # Verify expected strings in output
  - id: expected-output-check
    prompt: "Write a script that prints 'PASS: all tests green'."
    correctness:
      output:
        contains:
          - "PASS"
          - "all tests green"

  # Verify with a custom script
  - id: script-check
    prompt: "Create output.txt containing exactly 'hello world'."
    correctness:
      script: "./scripts/verify-output.sh"

  # Verify HTTP state
  - id: state-check
    prompt: "Start a web server on port 8080 that responds to GET /health with 'ok'."
    correctness:
      state:
        - url: "http://localhost:8080/health"
          method: GET
          expect: "ok"

  # Verify with an LLM judge
  - id: judge-check
    prompt: "Write a README.md explaining how to set up a Go project."
    correctness:
      judge:
        - "Does the README explain how to initialize a Go module?"
        - "Does it include instructions for running tests?"

  # Combine multiple checks
  - id: combined-check
    prompt: "Write calc.py that reads two numbers and prints their sum."
    correctness:
      agent_exits_ok: true
      output:
        contains:
          - "Result:"
      script: "./scripts/verify-calc.sh"
```

[View source](https://github.com/driangle/skival/tree/main/examples/correctness)

## Setup Hooks

Lifecycle hooks for fixture creation and cleanup: `before` runs once at the start, `reset` runs between samples, `after` runs once at the end.

```yaml
evals:
  - id: with-hooks
    prompt: "Read input.txt and write its contents reversed to output.txt."
    isolate: true
    setup:
      before: |
        echo "dlrow olleh" > input.txt
      reset: |
        rm -f output.txt
      after: |
        rm -f input.txt output.txt
    correctness:
      agent_exits_ok: true
```

[View source](https://github.com/driangle/skival/tree/main/examples/setup-hooks)

## Complexity Levels

Tag evals by difficulty and adjust sample counts and timeouts accordingly.

```yaml
evals:
  - id: low-complexity
    prompt: "Create a file called hello.txt containing 'hello'."
    complexity: low
    samples: 5
    timeout: 30

  - id: medium-complexity
    prompt: "Write a Python Flask app with a GET /users endpoint."
    complexity: medium
    samples: 3
    timeout: 120

  - id: high-complexity
    prompt: "Build a complete TODO app with SQLite, CRUD, and validation."
    complexity: high
    samples: 2
    timeout: 300
```

[View source](https://github.com/driangle/skival/tree/main/examples/complexity)

## Multiple Treatments

Compare a baseline control against multiple variations with different models, skills, environment variables, or runner configs.

```yaml
evals:
  - id: sort-algorithm
    prompt: "Write sort.py that reads integers from stdin and prints them sorted."
    model: "claude-sonnet-4-6"
    treatments:
      control:
        name: baseline
      variations:
        - name: with-skill
          skill: "./skills/python-best-practices.md"
        - name: opus-model
          model: "claude-opus-4-6"
        - name: with-env
          env:
            STYLE: "functional"
        - name: custom-runner-config
          runner_config:
            max_turns: 5
            allowed_tools: [Read, Write, Bash]
```

[View source](https://github.com/driangle/skival/tree/main/examples/multi-treatment)

## Multi-Runner

Compare different runners (claude-code, codex, aider) in the same suite.

```yaml
evals:
  - id: cross-runner
    prompt: "Write primes.py that prints all primes less than 100."
    correctness:
      agent_exits_ok: true
    treatments:
      control:
        name: claude-code
        model: "claude-sonnet-4-6"
        runner: claude-code
      variations:
        - name: codex
          model: "gpt-4.1"
          runner: codex
        - name: aider
          model: "claude-sonnet-4-6"
          runner: aider
```

[View source](https://github.com/driangle/skival/tree/main/examples/multi-runner)

## Matrix Comparison

Use `matrix` instead of `treatments` to generate cross-product combinations from multiple dimensions.

```yaml
evals:
  - id: hello-world
    prompt: "Write hello.sh that prints 'Hello, World!' to stdout."
    correctness:
      script: "./verify.sh"
    matrix:
      dimensions:
        - name: runner
          values:
            - label: claude-code
              runner: claude-code
            - label: codex
              runner: codex
        - name: model
          values:
            - label: opus
              model: claude-opus-4-6
            - label: sonnet
              model: claude-sonnet-4-6
```

This generates four treatments: `claude-code × opus`, `claude-code × sonnet`, `codex × opus`, `codex × sonnet`.

[View source](https://github.com/driangle/skival/tree/main/examples/matrix-comparison)

## Skillset Comparison

Compare no skill vs. a single skill vs. a composed skillset using `skills` (plural).

```yaml
evals:
  - id: fizzbuzz-skillset
    prompt: "Write fizzbuzz.sh that prints FizzBuzz for 1-20."
    treatments:
      control:
        name: "baseline"
      variations:
        - name: "shell-only"
          skill: "./skills/shell-best-practices.md"
        - name: "shell-and-testing"
          skills:
            - "./skills/shell-best-practices.md"
            - "./skills/testing-guidelines.md"
```

[View source](https://github.com/driangle/skival/tree/main/examples/skillset-comparison)

## Runner Config Precedence

Runner config merges at three levels: defaults → eval → treatment. Each level overrides the one above.

```yaml
defaults:
  runner: claude-code
  runner_config:
    max_turns: 20
    allowed_tools: [Read, Write, Bash, Glob, Grep]

evals:
  - id: eval-override
    prompt: "Write a test suite for a calculator module."
    runner_config:
      max_turns: 30             # overrides default
      permission_mode: "plan"   # new key, merged with defaults

  - id: treatment-override
    prompt: "Refactor the utils module."
    runner_config:
      max_turns: 25
    treatments:
      control:
        name: baseline
      variations:
        - name: restricted
          runner_config:
            max_turns: 10       # overrides eval's 25
            allowed_tools: [Read, Edit]
```

[View source](https://github.com/driangle/skival/tree/main/examples/runner-config)

## Per-Treatment Config

Override prompts and `config_dir` per treatment for full control over Claude Code settings, hooks, and MCP configuration.

```yaml
evals:
  - id: prompt-comparison
    treatments:
      control:
        name: baseline
        prompt: "Write a function that checks if a string is a palindrome."
      variations:
        - name: with-tests
          prompt: "Write a palindrome checker with comprehensive pytest tests."

  - id: config-comparison
    prompt: "List the files in the current directory"
    treatments:
      control:
        name: strict-config
        config_dir: "./configs/strict"
      variations:
        - name: permissive-config
          config_dir: "./configs/permissive"
```

[View source](https://github.com/driangle/skival/tree/main/examples/per-treatment-config)

## FizzBuzz

A complete real-world example with a skill file and verification script.

```yaml
version: 1
description: "FizzBuzz benchmark — compare baseline vs shell best-practices skill"

defaults:
  model: "claude-sonnet-4-6"
  samples: 3
  timeout: 60

evals:
  - id: fizzbuzz-basic
    prompt: |
      Write fizzbuzz.sh that prints FizzBuzz output for numbers 1 through 20.
      Rules: "Fizz" for multiples of 3, "Buzz" for 5,
      "FizzBuzz" for both, the number otherwise.
    complexity: low
    setup:
      reset: "rm -f fizzbuzz.sh"
    correctness:
      agent_exits_ok: true
      script: "./verify.sh"
    treatments:
      control:
        name: "baseline"
      variations:
        - name: "with-shell-skill"
          skill: "./skills/shell-best-practices.md"
```

[View source](https://github.com/driangle/skival/tree/main/examples/fizzbuzz)
