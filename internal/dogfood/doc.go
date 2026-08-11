// Package dogfood holds the deterministic self-tests for skival's own skill.
//
// skival ships a Claude Code skill (claude-code-plugin/skills/skival/SKILL.md)
// that teaches an agent to author suite.yaml files, plus a canonical dogfood
// suite (evals/suite.yaml) that evaluates whether injecting that skill
// improves an agent's output. The agent evaluation itself is expensive and
// non-deterministic, so it runs only in an opt-in workflow — but the harness
// around it must never rot. The tests in this package run in the normal
// `go test ./...` path and guarantee, with no LLM calls, that the skill file is
// well-formed and that the canonical suite still loads and validates against the
// current schema. See evals/README.md for the full picture.
package dogfood
