# Dogfood: skival evaluating its own skill

skival ships a Claude Code skill — [`claude-code-plugin/skills/skival/SKILL.md`](../claude-code-plugin/skills/skival/SKILL.md)
— that teaches an agent how to author `suite.yaml` files. This directory uses
skival to evaluate that skill: **does injecting the skill actually help an agent
produce a valid suite?**

Each eval asks an agent to write a `suite.yaml` for a task described *without*
the schema, then measures correctness by running `skival validate` on whatever
the agent produced. The `with-skill` variant injects the real `SKILL.md`, so the
skill is the only source of schema knowledge — that's what makes the baseline vs.
with-skill comparison meaningful.

## Two layers

Dogfooding the skill means running a real agent, which is **non-deterministic,
costs money, and needs an API key**. So the harness is split in two:

| Layer | Runs | What it guarantees | Cost |
|-------|------|--------------------|------|
| **Deterministic** (`internal/dogfood`, `internal/suite/skilldoc_test.go`) | Every PR, via `go test ./...` | The skill file is well-formed, `evals/suite.yaml` still loads/validates against the current schema, and the skill documents exactly the verifier types the validator accepts. **No LLM calls.** | Free |
| **Agent eval** (`.github/workflows/dogfood.yaml`) | Opt-in only (manual `workflow_dispatch`) | Whether injecting the skill measurably improves agent output. Invokes the `claude-code` runner. | LLM $$$ |

The deterministic layer keeps the harness honest so it never rots between agent
runs; the agent-eval layer is the actual measurement, run on demand.

## Files

- `suite.yaml` — the canonical dogfood suite: three evals (model-comparison,
  skill-ab, tool-access), each `baseline` vs. `with-skill`.
- `prompts/*.md` — self-contained task prompts that describe *what* to evaluate
  but deliberately omit the skival schema.
- `verify.sh` — correctness check: runs `skival validate` on the agent's produced
  `suite.yaml` (wired as a `check_output` verifier).
- `workdir/<eval-id>/` — per-eval working directory where the agent writes its
  `suite.yaml` (git-ignored; only `.gitkeep` is tracked).

## Running the agent eval

Locally (needs `skival` and the `claude` CLI on PATH, and `ANTHROPIC_API_KEY`):

```bash
make install                 # ensure the current skival is on PATH
skival validate evals/suite.yaml
skival run evals/suite.yaml --results-dir results/dogfood
```

To read the agent transcript behind each run, add `--link-sessions --format html`;
skival writes a `report.html` into the results dir whose per-run "view session"
links open the rendered conversation:

```bash
skival run evals/suite.yaml --results-dir results/dogfood --format html --link-sessions
open results/dogfood/<run>/report.html
```

In CI: open the **Actions → Dogfood (skill eval)** workflow and click *Run
workflow*. It requires an `ANTHROPIC_API_KEY` repository secret and uploads the
report and results as an artifact. It never runs on push/PR and never gates a
merge.
