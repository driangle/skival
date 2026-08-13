# CLI Reference

## Global Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable debug-level logging |

## `skival run`

Execute an eval suite.

```bash
skival run <suite.yaml> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--samples N` | `1` | Number of runs per variant |
| `-p, --parallel N` | `0` | Max concurrent samples (`0` or `1` = sequential) |
| `--results-dir <path>` | | Save results to disk for later reporting |
| `--variants <names>` | | Comma-separated list of variant names to run |
| `--evals <ids>` | | Comma-separated list of eval IDs to run |
| `--format <type>` | `markdown` | Output format: `markdown`, `json`, or `html` |
| `--timeout <secs>` | | Timeout in seconds for all evals (overrides suite/eval-level timeouts) |
| `--compare` | | Force comparative judging on where criteria are configured (overrides `enabled: false`) |
| `--no-compare` | | Disable comparative judging even if the suite configures it |

`--compare` and `--no-compare` are mutually exclusive. When neither is given, comparative judging follows the suite's [`compare`](/configuration#comparative-judging) configuration.

### Examples

Run all evals:

```bash
skival run suite.yaml
```

Run specific evals with 5 samples:

```bash
skival run suite.yaml --evals fizzbuzz,sorting --samples 5
```

Run only the baseline variant and save results:

```bash
skival run suite.yaml --variants baseline --results-dir ./results
```

Run with 4 concurrent samples:

```bash
skival run suite.yaml --samples 10 --parallel 4
```

Override timeout to 2 minutes for all evals:

```bash
skival run suite.yaml --timeout 120
```

Run a cheap pass without comparative judging, even if the suite configures it:

```bash
skival run suite.yaml --no-compare
```

### Ranking

Both `skival run` and `skival report` rank variants by a composite score. The weighting lives in the suite's [`ranking`](/configuration#ranking) block, not on the command line. By default the composite weights correctness, cost, and duration. When a suite's runner reports no cost (local models, self-hosted models, the `exec` runner), set `ranking.weights.tokens` to rank on model-agnostic token efficiency instead of `cost` — see [Ranking on tokens instead of cost](/configuration#ranking-on-tokens-instead-of-cost). Weight `cost` **or** `tokens`, not both: they measure the same economic signal and summing them double-counts it.

## `skival report`

Regenerate reports from previously saved results.

```bash
skival report <results-dir> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format <type>` | `markdown` | Output format: `markdown`, `json`, or `html` |

### Example

```bash
skival report ./results --format json
```

## `skival validate`

Parse and validate a suite file without executing it. Reports the suite structure including version, description, eval count, and variant configuration.

```bash
skival validate <suite.yaml>
```

### Example

```bash
skival validate suite.yaml
```

Output:

```
Suite is valid
  Version:     1
  Description: My eval suite
  Evals:       3
  ...
```

## `skival compare`

Compare two result directories and produce a diff report showing how variants changed between runs. Useful for seeing what improved, regressed, or stayed the same after tweaking skills or prompts.

```bash
skival compare <baseline-dir> <candidate-dir> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format <type>` | `markdown` | Output format: `markdown`, `json`, or `html` |

### Examples

Compare two runs:

```bash
skival compare results/run-1 results/run-2
```

Output as JSON for programmatic consumption:

```bash
skival compare results/run-1 results/run-2 --format json
```

### Output

The report shows per-eval, per-variant deltas for:

- **Pass rate** — percentage point change (e.g. `+50pp ↑`)
- **Median cost** — absolute USD and percentage change (e.g. `-$0.0200 (-40.0%) ↓`)
- **Median duration** — absolute and percentage change (e.g. `-2.0s (-20.0%) ↓`)

Variants that exist in only one run are labeled `added` or `removed` rather than causing errors.

Returns a non-zero exit code if either directory is missing or invalid.

## `skival version`

Print the current skival version.

```bash
skival version
```
