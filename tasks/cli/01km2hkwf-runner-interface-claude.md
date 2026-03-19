---
title: "Runner interface and Claude implementation"
id: "01km2hkwf"
status: pending
priority: high
type: feature
tags: ["phase-1", "runner"]
dependencies: ["01km2hk6e"]
created: "2026-03-19"
phase: phase-1
---

# Runner interface and Claude implementation

## Objective

Define the Runner interface and implement ClaudeRunner using `github.com/driangle/agentrunner-go` to execute prompts via the Claude Code CLI.

## Tasks

- [ ] Define Runner interface in `internal/runner/runner.go` with RunInput/RunOutput/Metrics types
- [ ] Add `agentrunner-go` as a Go module dependency
- [ ] Implement ClaudeRunner in `internal/runner/claude.go` that maps RunInput to agentrunner options and RunOutput from agentrunner Result
- [ ] Map treatment config fields (model, allowed_tools, env, working_dir, system_prompt, mcp_config) to agentrunner options
- [ ] Extract metrics (tokens, cost, duration, session ID) from agentrunner Result into Metrics struct
- [ ] Handle errors and timeouts gracefully

## Acceptance Criteria

- Runner interface is clean and agent-agnostic (no Claude-specific types in the interface)
- ClaudeRunner correctly invokes claude CLI via agentrunner and returns populated RunOutput
- All treatment override fields are passed through to the CLI
- Errors from the CLI are wrapped and returned, not swallowed
