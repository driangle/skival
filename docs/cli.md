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
| `--samples N` | `1` | Number of runs per treatment |
| `-p, --parallel N` | `0` | Max concurrent samples (`0` or `1` = sequential) |
| `--results-dir <path>` | | Save results to disk for later reporting |
| `--treatments <names>` | | Comma-separated list of treatment names to run |
| `--evals <ids>` | | Comma-separated list of eval IDs to run |
| `--format <type>` | `markdown` | Output format: `markdown` or `json` |

### Examples

Run all evals:

```bash
skival run suite.yaml
```

Run specific evals with 5 samples:

```bash
skival run suite.yaml --evals fizzbuzz,sorting --samples 5
```

Run only the control treatment and save results:

```bash
skival run suite.yaml --treatments control --results-dir ./results
```

Run with 4 concurrent samples:

```bash
skival run suite.yaml --samples 10 --parallel 4
```

## `skival report`

Regenerate reports from previously saved results.

```bash
skival report <results-dir> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format <type>` | `markdown` | Output format: `markdown` or `json` |

### Example

```bash
skival report ./results --format json
```

## `skival validate`

Parse and validate a suite file without executing it. Reports the suite structure including version, description, eval count, and treatment configuration.

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

Compare two result directories and produce a diff report showing how treatments changed between runs. Useful for seeing what improved, regressed, or stayed the same after tweaking skills or prompts.

```bash
skival compare <baseline-dir> <candidate-dir> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format <type>` | `markdown` | Output format: `markdown` or `json` |

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

The report shows per-eval, per-treatment deltas for:

- **Pass rate** — percentage point change (e.g. `+50pp ↑`)
- **Median cost** — absolute USD and percentage change (e.g. `-$0.0200 (-40.0%) ↓`)
- **Median duration** — absolute and percentage change (e.g. `-2.0s (-20.0%) ↓`)

Treatments that exist in only one run are labeled `added` or `removed` rather than causing errors.

Returns a non-zero exit code if either directory is missing or invalid.

## `skival version`

Print the current skival version.

```bash
skival version
```
