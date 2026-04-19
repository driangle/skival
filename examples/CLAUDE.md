# Examples

Each example lives in its own directory with a `suite.yaml` entry point plus any supporting files (skills, scripts, referenced evals).

```
examples/
  my-example/
    suite.yaml              # required — the suite definition
    skills/                 # skill files referenced by treatments
    scripts/                # verification scripts for correctness.script
    evals/                  # eval files referenced via file:
```

## Requirements

- Every example must pass `Load()` — the `TestLoad_Examples` smoke test in `internal/suite/loader_test.go` automatically picks up all `examples/*/suite.yaml` files.
- Examples must be self-contained: any file referenced by the suite (skills, scripts, eval files) must exist in the same directory.
- Set `model` at the defaults, eval, or variant level — validation requires every variant to resolve to a model.
- Keep examples focused on one concept each and add YAML comments where the behavior isn't obvious.
