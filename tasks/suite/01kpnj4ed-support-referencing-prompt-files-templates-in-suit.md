---
title: "Support referencing prompt files/templates in suite.yaml"
id: "01kpnj4ed"
status: pending
priority: medium
type: feature
tags: ["yaml-api", "prompt"]
created: "2026-04-20"
---

# Support referencing prompt files/templates in suite.yaml

## Objective

Allow the `prompt` field in a `suite.yaml` (and in standalone eval files loaded via `file:`) to be sourced from a separate file instead of being inlined as a YAML string. Long prompts bloat `suite.yaml`, lose syntax highlighting, and are awkward to diff or reuse across evals/variants. A `promptFile:` (or equivalent) reference keeps the suite definition focused on configuration while the prompt lives in a plain text/markdown file.

Optionally, support simple variable substitution inside the referenced file (e.g. `{{var}}` placeholders filled from a `vars:` map on the eval/variant) so a single template can be parameterized across variants without duplication.

### Example target shape

```yaml
evals:
  - id: fibonacci
    promptFile: prompts/fibonacci.md
    variants:
      - name: baseline
      - name: opus
        model: claude-opus-4-6
```

With optional variable support:

```yaml
evals:
  - id: refactor
    promptFile: prompts/refactor.md
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

Where `prompts/refactor.md` contains `{{language}}` / `{{tone}}` placeholders.

## Tasks

- [ ] Decide on field name (`promptFile` vs `prompt_file`) and document precedence rules (error if both `prompt` and `promptFile` are set on the same level)
- [ ] Add `PromptFile string` to `Eval` and `Treatment` (variant) structs in `internal/suite/suite.go`
- [ ] Resolve `promptFile` paths relative to the suite file (and relative to the eval file when loaded via `file:`), consistent with existing file-relative resolution for skills/scripts
- [ ] Load file contents in the loader and populate the resolved `Prompt` field so the executor sees no behavior change
- [ ] Validate that referenced files exist and are readable during `Load()` / `validate` command
- [ ] Evaluate whether variable substitution is worth including in v1 — if yes:
  - [ ] Add `vars: map[string]string` on `Eval` and `Treatment`; merge variant vars over eval vars
  - [ ] Implement a minimal `{{name}}` substitution pass over the loaded prompt template
  - [ ] Error on unresolved placeholders (strict mode) to avoid silent typos
- [ ] Update `apps/cli/cmd/validate.go` to surface `promptFile` resolution errors clearly
- [ ] Add loader unit tests covering: happy path, missing file, both `prompt` + `promptFile` set, per-variant override, path resolution from nested `file:` evals, and (if implemented) var substitution
- [ ] Add an `examples/prompt-file/` suite demonstrating the feature so `TestLoad_Examples` exercises it
- [ ] Update `docs/configuration.md` and `docs/examples.md` with the new field and a short template example

## Acceptance Criteria

- An eval can reference an external prompt file via `promptFile:` at the eval or variant level, and the runner uses the file's contents as the prompt
- Paths resolve relative to the containing suite/eval file, matching how skills and scripts already resolve
- Loader errors clearly when the referenced file is missing or when both `prompt` and `promptFile` are set on the same level
- Existing suites using inline `prompt:` continue to work unchanged
- If variable substitution is included: placeholders defined in `vars` are substituted in the loaded prompt, variant vars override eval vars, and unresolved placeholders fail loading
- New example suite passes `TestLoad_Examples`
- Documentation and tests cover the new field
