---
id: "01kz6nr0f"
title: "Align documentation with the flat variants schema"
status: pending
priority: critical
effort: medium
type: docs
tags: ["docs", "yaml-api"]
created: 2026-08-04
---

# Align documentation with the flat variants schema

## Objective

Every YAML example in `README.md`, `docs/getting-started.md`, and
`docs/configuration.md` still uses the removed `treatments:` / `control:` /
`variations:` schema and a non-existent `--treatments` flag. The code was
migrated to a flat `variants:` list and a `--variants` flag in task
`01kpj1s17` (completed), but the docs were never updated. Because the loader
uses non-strict YAML, the unknown `treatments` key is silently dropped, so the
documented Quick Start produces an eval with zero variants.

Verified — the README Quick Start fails against the current binary:

```
$ skival validate readme-quickstart.yaml
validation failed with 1 error(s):
- eval[0]: at least one variant is required
```

The committed `examples/*/suite.yaml` already use the correct `variants:`
syntax, so they are the reference for what the docs should say.

## Tasks

- [ ] Replace `treatments/control/variations` with a flat `variants:` list in
      `README.md` (Quick Start + Features + Key Flags table)
- [ ] Change the documented filter flag from `--treatments` to `--variants`
- [ ] Update `docs/getting-started.md` examples and prose (control/variation wording)
- [ ] Update `docs/configuration.md` examples (incl. the matrix section that says
      "The first combination becomes the control")
- [ ] Replace the deprecated `correctness:` block in the Quick Start with a
      `verify:` list (the current example also triggers a deprecation warning)
- [ ] Decide the `treatments`→`variants` back-compat story: task `01kpj1s17`
      claimed old `treatments` YAML still loads via migration, but no such
      migration exists in `internal/suite/loader.go` (validate rejects it).
      Either restore the migration or drop that claim from the docs.
- [ ] Add `make validate-examples`-style coverage or a doc-snippet test so doc
      YAML can't drift from the schema again

## Acceptance Criteria

- Copy-pasting the README Quick Start passes `skival validate` and runs
- No occurrence of `treatments:`, `control:`, `variations:`, or `--treatments`
  remains in `README.md` or `docs/`
- Documented examples use `verify:`, not the deprecated `correctness:` field
