# Examples

The [examples/](https://github.com/driangle/skival/tree/main/examples) directory contains self-contained suites that demonstrate each skival feature. Clone the repo and run any example directly:

```bash
skival run examples/minimal/suite.yaml
```

## Minimal

The simplest valid suite — one eval, two variants.

```yaml
version: 1

defaults:
  runner: claude-code

evals:
  - id: hello
    dir: "./workdir"
    prompt: "Write a file called hello.txt containing 'Hello, World!'"
    verify:
      - type: file_contains
        path: hello.txt
        contains: "Hello, World!"
    variants:
      - name: baseline
        model: "claude-sonnet-4-6"
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
    dir: "./workdir"
    prompt: "Create a file called greeting.txt with a friendly greeting."
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
      - name: fewer-turns
        runner_config:
          max_turns: 5

  - id: overrides-defaults
    dir: "./workdir"
    prompt: "Write a script that prints the current date."
    samples: 3        # overrides suite default
    timeout: 60       # overrides suite default
    model: "claude-opus-4-6"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
      - name: sonnet-model
        model: "claude-sonnet-4-6"
```

[View source](https://github.com/driangle/skival/tree/main/examples/defaults)

## File References

Split eval definitions into separate files to keep large suites organized.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  samples: 2

evals:
  - file: evals/string-reverse.yaml
  - file: evals/fibonacci.yaml
```

[View source](https://github.com/driangle/skival/tree/main/examples/file-refs)

## Prompt Files

Source prompts from external files instead of inlining them, and parameterize a template across variants with `{{var}}` substitution.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"

evals:
  - id: refactor
    prompt_file: prompts/refactor.md   # contains {{language}} / {{tone}}
    vars:
      language: "Go"
    variants:
      - name: strict
        vars:
          tone: "terse and precise"
      - name: verbose
        vars:
          tone: "detailed and explanatory"
```

[View source](https://github.com/driangle/skival/tree/main/examples/prompt-file)

## Verification

Every verification mode in one suite. Each eval declares a `verify:` list of steps — a step type such as `check`, `agent_exits_ok`, `output_contains`, `check_output`, `http_check`, `file_contains`, `command`, `tcp_check`, or `judge`.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"

evals:
  # Verify via a shell command
  - id: check-verify
    dir: "./workdir"
    prompt: "Write a Go program in main.go that prints 'hello'."
    verify:
      - type: check
        run: "go build ./..."
    variants:
      - name: baseline

  # Verify the agent exits successfully
  - id: agent-exits-ok-check
    dir: "./workdir"
    prompt: "Write a shell script run.sh that exits with code 0."
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline

  # Verify expected strings appear in the agent's output
  - id: expected-output-check
    dir: "./workdir"
    prompt: "What is 7 * 6? Reply with just the number."
    verify:
      - type: output_contains
        values:
          - "42"
    variants:
      - name: baseline

  # Verify by piping agent output to a script's stdin
  - id: check-output-verify
    dir: "./workdir"
    prompt: "Create output.txt containing exactly 'hello world'."
    verify:
      - type: check_output
        run: "./scripts/verify-output.sh"
    variants:
      - name: baseline

  # Verify an HTTP endpoint
  - id: http-check
    dir: "./workdir"
    prompt: "Start a web server on port 8080 that responds to GET /health with 'ok'."
    verify:
      - type: http_check
        url: "http://localhost:8080/health"
        method: GET
        status: 200
        body_contains: "ok"
    variants:
      - name: baseline

  # Verify a file on disk
  - id: file-contains-check
    dir: "./workdir"
    prompt: "Create output.txt containing 'hello world'."
    verify:
      - type: file_contains
        path: "output.txt"
        exists: true
        contains: "hello world"
    variants:
      - name: baseline

  # Verify with an LLM judge
  - id: judge-check
    dir: "./workdir"
    prompt: "Write a README.md explaining how to set up a Go project."
    verify:
      - type: judge
        criteria:
          - "Does the README explain how to initialize a Go module?"
          - "Does it include instructions for running tests?"
    variants:
      - name: baseline

  # Combine multiple checks
  - id: combined-check
    dir: "./workdir"
    prompt: "Write calc.py that reads two numbers and prints their sum prefixed with 'Result:'."
    verify:
      - type: agent_exits_ok
      - type: output_contains
        values:
          - "Result:"
      - type: check_output
        run: "./scripts/verify-calc.sh"
    variants:
      - name: baseline
```

[View source](https://github.com/driangle/skival/tree/main/examples/correctness)

## Setup Hooks

Lifecycle hooks for fixture creation and cleanup: `before` runs once at the start, `reset` runs between samples, `after` runs once at the end.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"

evals:
  - id: with-hooks
    dir: "./workdir"
    prompt: "Read input.txt and write its contents reversed to output.txt."
    isolate: true
    setup:
      before: |
        echo "dlrow olleh" > input.txt
      reset: |
        rm -f output.txt
      after: |
        rm -f input.txt output.txt
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
```

[View source](https://github.com/driangle/skival/tree/main/examples/setup-hooks)

## Complexity Levels

Tag evals by difficulty and adjust sample counts and timeouts accordingly.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"

evals:
  - id: low-complexity
    dir: "./workdir"
    prompt: "Create a file called hello.txt containing 'hello'."
    samples: 5
    timeout: 30
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline

  - id: medium-complexity
    dir: "./workdir"
    prompt: "Write a Python Flask app with a GET /users endpoint."
    samples: 3
    timeout: 120
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline

  - id: high-complexity
    dir: "./workdir"
    prompt: "Build a complete TODO app with SQLite, CRUD, and validation."
    samples: 2
    timeout: 300
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
```

[View source](https://github.com/driangle/skival/tree/main/examples/complexity)

## Multiple Variants

Compare a baseline against multiple variants with different models, skills, environment variables, or runner configs.

```yaml
version: 1

defaults:
  runner: claude-code

evals:
  - id: sort-algorithm
    dir: "./workdir"
    prompt: "Write sort.py that reads integers from stdin and prints them sorted."
    model: "claude-sonnet-4-6"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
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
version: 1

defaults:
  samples: 2
  timeout: 120

evals:
  - id: cross-runner
    dir: "./workdir"
    prompt: "Write primes.py that prints all primes less than 100."
    verify:
      - type: agent_exits_ok
    variants:
      - name: claude-code
        model: "claude-sonnet-4-6"
        runner: claude-code
      - name: codex
        model: "gpt-4.1"
        runner: codex
      - name: aider
        model: "claude-sonnet-4-6"
        runner: aider
```

[View source](https://github.com/driangle/skival/tree/main/examples/multi-runner)

## Matrix Comparison

Use `matrix` instead of `variants` to generate cross-product combinations from multiple dimensions.

```yaml
version: 1

defaults:
  runner: claude-code

evals:
  - id: hello-world
    dir: "./workdir"
    prompt: "Write hello.sh that prints 'Hello, World!' to stdout."
    verify:
      - type: check_output
        run: "./verify.sh"
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

This generates four variants: `claude-code × opus`, `claude-code × sonnet`, `codex × opus`, `codex × sonnet`.

[View source](https://github.com/driangle/skival/tree/main/examples/matrix-comparison)

## Skillset Comparison

Compare no skill vs. a single skill vs. a composed skillset using `skills` (plural).

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"

evals:
  - id: fizzbuzz-skillset
    dir: "./workdir"
    prompt: "Write fizzbuzz.sh that prints FizzBuzz for 1-20."
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
      - name: shell-only
        skill: "./skills/shell-best-practices.md"
      - name: shell-and-testing
        skills:
          - "./skills/shell-best-practices.md"
          - "./skills/testing-guidelines.md"
```

[View source](https://github.com/driangle/skival/tree/main/examples/skillset-comparison)

## Runner Config Precedence

Runner config merges at three levels: defaults → eval → variant. Each level overrides the one above.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  runner_config:
    max_turns: 20
    allowed_tools: [Read, Write, Bash, Glob, Grep]

evals:
  - id: eval-override
    dir: "./workdir"
    prompt: "Write a test suite for a calculator module."
    verify:
      - type: agent_exits_ok
    runner_config:
      max_turns: 30             # overrides default
      permission_mode: "plan"   # new key, merged with defaults
    variants:
      - name: baseline

  - id: variant-override
    dir: "./workdir"
    prompt: "Refactor the utils module."
    verify:
      - type: agent_exits_ok
    runner_config:
      max_turns: 25
    variants:
      - name: baseline
      - name: restricted
        runner_config:
          max_turns: 10         # overrides eval's 25
          allowed_tools: [Read, Edit]
```

[View source](https://github.com/driangle/skival/tree/main/examples/runner-config)

## Per-Variant Config

Override prompts and `config_dir` per variant for full control over Claude Code settings, hooks, and MCP configuration.

```yaml
version: 1

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"

evals:
  - id: prompt-comparison
    dir: "./workdir"
    verify:
      - type: agent_exits_ok
    variants:
      - name: baseline
        prompt: "Write a function that checks if a string is a palindrome."
      - name: with-tests
        prompt: "Write a palindrome checker with comprehensive pytest tests."

  - id: config-comparison
    dir: "./workdir"
    prompt: "List the files in the current directory"
    verify:
      - type: agent_exits_ok
    variants:
      - name: strict-config
        config_dir: "./configs/strict"
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
  runner: claude-code
  model: "claude-sonnet-4-6"
  samples: 3
  timeout: 60

evals:
  - id: fizzbuzz-basic
    dir: "./workdir"
    prompt: |
      Write fizzbuzz.sh that prints FizzBuzz output for numbers 1 through 20.
      Rules: "Fizz" for multiples of 3, "Buzz" for 5,
      "FizzBuzz" for both, the number otherwise.
    setup:
      reset: "rm -f fizzbuzz.sh"
    verify:
      - type: agent_exits_ok
      - type: check_output
        run: "./verify.sh"
    variants:
      - name: baseline
      - name: with-shell-skill
        skill: "./skills/shell-best-practices.md"
```

[View source](https://github.com/driangle/skival/tree/main/examples/fizzbuzz)

## Comparative Judging

Score the outputs of the variants that pass an eval against each other on a 1–5 quality scale, and feed the result into ranking through the `quality` weight. Comparison runs only as a tiebreaker among passing variants — see [Comparative Judging](/configuration#comparative-judging).

```yaml
version: 1
description: "Compare explanation quality across variants that all pass"

defaults:
  runner: claude-code
  model: "claude-sonnet-4-6"
  judge_model: "claude-haiku-4-5-20251001"

compare:
  criteria:
    - "explanation is clear and easy to follow"
    - "covers the key trade-offs"
  weight: 0.2

evals:
  - id: explain-quicksort
    prompt: "Explain how quicksort works and its time complexity."
    verify:
      - type: judge
        criteria:
          - "correctly describes partitioning around a pivot"
          - "states average O(n log n) and worst-case O(n^2)"
    variants:
      - name: concise
        prompt: "Explain quicksort and its time complexity in under 150 words."
      - name: detailed
        prompt: "Explain quicksort in depth, with pivot selection and an example."
```

[View source](https://github.com/driangle/skival/tree/main/examples/comparative-judging)
