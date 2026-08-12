---
title: "Migrate session linking from vibeview CLI to Go SDK"
id: "01kztjxep"
status: completed
priority: medium
type: chore
tags: ["reporting", "vibeview", "sessions", "sdk"]
created: "2026-08-12"
completed_at: 2026-08-12
---

# Migrate session linking from vibeview CLI to Go SDK

## Description

Session linking ([[01kzsr2nq]]) currently shells out to the `vibeview` CLI from
`internal/sessionlink`, which means **end users must have `vibeview` installed on
PATH** — otherwise it falls back to a `vibeview show <id>` hint. Once vibeview
publishes a public Go render package + tagged release (vibeview task `01kztbrsq`),
switch to **importing the SDK** so the renderer is compiled into skival and no
external binary is required.

**Unblocked** as of vibeview `apps/lib/v0.2.0` (task `01kztbrsq`). The SDK is:

```go
import "github.com/driangle/vibeview/apps/lib/sessionhtml"

// Session accepts a .jsonl path (our sidecar) or a session id.
html, err := sessionhtml.RenderSessionHTML(sessionhtml.Request{
    Session:     sidecarPath,
    CostEnabled: true,
})
```

`RenderSessionHTML` returns the complete self-contained HTML document as bytes —
skival writes them to `run-N.session.html`.

## Tasks

- [x] `go get github.com/driangle/vibeview/apps/lib@v0.2.0` and pin it in `go.mod`.
- [x] Replace the `exec.Command` call in `internal/sessionlink/sessionlink.go`
      with `sessionhtml.RenderSessionHTML(...)`, writing the bytes to the same
      `run-N.session.html`. Drop the `exec.LookPath`/binary detection.
- [x] Decide the missing-renderer story now that it's compile-time: with the SDK
      linked in there is no "absent binary" case, so `--link-sessions` always
      produces pages (keep the fallback only for genuine render errors).
- [x] Update `internal/sessionlink` tests: drop the fake-`vibeview`-on-PATH shim
      in favor of exercising the SDK directly (the persist round-trip test can
      then use real rendering instead of a stub).
- [x] Refresh docs: `--link-sessions` no longer needs `vibeview` installed; note
      the pinned SDK version instead (`README.md`, `evals/README.md`).

## Acceptance Criteria

- `skival run --link-sessions` renders session pages with **no `vibeview` binary
  on PATH** (SDK compiled in).
- `go.mod` pins the released vibeview render module; `make check` passes.
- Tests no longer depend on a fake `vibeview` executable.
- Docs updated to drop the runtime `vibeview` prerequisite.
