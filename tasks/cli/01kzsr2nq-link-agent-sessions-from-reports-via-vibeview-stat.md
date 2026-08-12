---
title: "Link agent sessions from reports via vibeview static export"
id: "01kzsr2nq"
status: completed
priority: medium
type: feature
tags: ["reporting", "vibeview", "sessions"]
created: "2026-08-12"
completed_at: 2026-08-12
---

# Link agent sessions from reports via vibeview static export

## Objective

Let a skival HTML report link out to the actual agent session (the full
transcript) behind each run, so a reviewer can go from "with-skill passed, cost
$0.14" straight into reading what the agent actually did. Sessions are rendered
by [vibeview](https://github.com/driangle/vibeview) as **static, self-contained
HTML** — no live server — so the linked report stays portable (works offline and
inside a CI artifact).

### What already exists (no work needed)

- skival persists each run's session two ways: `session_id` in `run-N.json`, and
  a portable sidecar `run-N.conversation.jsonl` next to the report.
- The sidecar is already in a format vibeview reads — `vibeview inspect
  <sidecar>` parses it cleanly (messages, tools, tokens, cost). No conversion.

### The missing pieces

1. **vibeview** has no static HTML export yet (only `show` text/JSON and a live
   `web` server) — see the companion task below.
2. **skival** doesn't invoke it or emit session links in the report.

This task covers the **skival side**. It depends on the vibeview export command
(tracked separately, drafted below) but degrades gracefully without it.

## Design

- Add an opt-in flag on `skival run` (e.g. `--link-sessions`) — off by default so
  the report has no hard dependency on vibeview.
- When enabled, after each sample skival shells out to
  `vibeview export <run-N.conversation.jsonl> --format html --out run-N.session.html`
  (writing next to the existing sidecars) and records the relative path on the
  run result.
- The **HTML report** renders a per-run "View session" link to that file when
  present. Markdown/JSON reports include the relative path / `session_id` too.
- **Graceful fallback:** if `vibeview` is not on PATH or export fails, skip the
  static page and instead surface a copy-pasteable hint
  (`vibeview show <session_id>`), plus the raw `session_id`, so the report is
  still useful. Never fail the run because linking failed.
- Keep the vibeview integration isolated behind a small helper (single
  responsibility, easy to stub in tests) rather than threading exec calls through
  the executor.

## Tasks

- [x] Add `--link-sessions` flag to `skival run` (default off); plumb an option
      down to the executor.
- [x] Add a small `sessionlink` helper that detects `vibeview` on PATH and shells
      out to `vibeview export … --format html --out …`; return the produced path
      or a fallback (session_id + `vibeview show` hint). No panic/hard-fail on
      error.
- [x] Persist the produced session-page relative path (and keep `session_id`) so
      `skival report` from disk can render the link without re-exporting.
- [x] Render "View session" links in the HTML report next to each run's
      cost/duration; include path/`session_id` in markdown & JSON reports.
- [x] Tests: helper returns the link when a fake `vibeview` succeeds and the
      fallback when it is absent/fails; report includes the link when present and
      the hint when not. Do not require a real vibeview binary in tests.
- [x] Document the flag and the vibeview prerequisite in `README.md` and
      `evals/README.md`.

## Dependency: vibeview static export (separate repo)

Draft for a task to file in the **vibeview** repo (skival shells out to this):

> **Add `vibeview export` — static single-session HTML.**
> `vibeview export <session-id|path.jsonl> --format html --out <file>` renders one
> session to a **self-contained** HTML file (inline CSS/JS, no external requests,
> no running server), reusing the existing SessionView rendering. Mirrors the
> planned "export SessionView to PDF" work. Acceptance: the output opens offline
> in a browser and shows the conversation, tool calls, tokens, and cost; exit
> non-zero with a clear message on an unknown session.

## Acceptance Criteria

- `skival run --link-sessions` produces a per-run static session HTML (via
  `vibeview export`) beside the sidecars, and the HTML report links to it.
- With `vibeview` absent or export failing, the run still succeeds and the report
  shows a `session_id` + `vibeview show` fallback instead of a broken link.
- The session link/path is persisted so `skival report <dir>` renders it without
  re-exporting.
- New tests cover the success and fallback paths without needing a real vibeview
  binary; `make check` passes.
- The flag and vibeview prerequisite are documented.
