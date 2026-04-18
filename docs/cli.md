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

## `skival version`

Print the current skival version.

```bash
skival version
```
