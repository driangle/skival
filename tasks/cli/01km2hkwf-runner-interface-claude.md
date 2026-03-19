---
title: "Runner interface and Claude implementation"
id: "01km2hkwf"
status: completed
priority: high
type: feature
tags: ["phase-1", "runner"]
dependencies: ["01km2hk6e"]
created: "2026-03-19"
phase: phase-1
---

# Runner interface and Claude implementation

## Objective

Define the Runner interface and implement ClaudeRunner using `github.com/driangle/agent-runner/go` to execute prompts via the Claude Code CLI.

### Dependency

```bash
go get github.com/driangle/agent-runner/go@v0.1.0
```

```go
import (
    agentrunner "github.com/driangle/agent-runner/go"
    "github.com/driangle/agent-runner/go/claudecode"
)
```

### API Reference

```go
runner := claudecode.NewRunner()

// Simple run
result, err := runner.Run(ctx, "prompt",
    agentrunner.WithMaxTurns(3),
    agentrunner.WithSkipPermissions(true),
    agentrunner.WithModel("claude-sonnet-4-6"),
)
// result.Text, result.CostUSD, result.SessionID

// Streaming
msgCh, errCh := runner.RunStream(ctx, "prompt", ...opts)

// Resume session
result2, _ := runner.Run(ctx, "follow-up",
    claudecode.WithResume(result.SessionID),
)
```

## Tasks

- [x] Add `github.com/driangle/agent-runner/go@v0.1.0` as a Go module dependency

## Decision

Dropped the `internal/runner` wrapper — the `agentrunner` library is used directly. The wrapper added minimal value (1:1 option pass-through, one-line `SkipPermissions`). A Runner abstraction can be introduced later if a second agent backend is needed.
