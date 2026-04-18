---
title: "Introduce typed correctness probes to replace state and unify assertion API"
id: "01kpgv3xy"
status: pending
priority: high
type: feature
tags: ["correctness", "yaml-api"]
created: "2026-04-18"
---

# Introduce typed correctness probes to replace state and unify assertion API

## Objective

Replace `correctness.state` (HTTP-only, implicit `expect` semantics) with a typed `correctness.probes` array that uses discriminator keys (`http`, `file`, `command`, `tcp`) and explicit `assert` blocks. While doing so, audit the full `correctness` object for consistency — several existing checks overlap with what probes could express (`script` ≈ `command` probe, `compiles` ≈ `command` probe, `agent_exits_ok` ≈ agent-level check that doesn't belong with external probes).

### Design constraints

- Each probe entry uses exactly one type key as discriminator (`http:`, `file:`, `command:`, `tcp:`)
- Every probe type has an `assert` sub-object with explicitly named checks — no implicit semantics
- `probes` replaces `state` entirely — `state` becomes a deprecated alias during migration
- Existing checks that overlap with probes should be evaluated for consolidation:
  - `script` → could become a `command` probe, but `script` receives agent output on stdin which is a different contract — keep `script` separate unless the command probe can also opt into stdin piping
  - `compiles` → conceptually a `command` probe with `exits: 0`, but it runs in the eval dir and is a well-understood shorthand — consider keeping as sugar
  - `agent_exits_ok` → checks the agent's own exit code, not external state — this is fundamentally different from probes and should stay separate
  - `output.contains` → checks agent stdout, not external state — keep separate
  - `judge` → LLM-based, orthogonal to probes — keep separate

### Proposed YAML shape

```yaml
correctness:
  # Agent-level checks (unchanged)
  agent_exits_ok: true
  output:
    contains: ["PASS"]
  compiles: "go build ./..."
  script: "./scripts/verify.sh"
  judge:
    - "Does the output explain X?"

  # External state probes (new, replaces `state`)
  probes:
    - http:
        url: "http://localhost:8080/health"
        method: GET
        assert:
          status: 200
          body_contains: "ok"

    - file:
        path: "output.txt"
        assert:
          exists: true
          contains: "hello world"

    - command:
        run: "pg_isready -h localhost"
        assert:
          exits: 0
          stdout_contains: "accepting connections"

    - tcp:
        host: "localhost"
        port: 5432
        assert:
          open: true
```

## Tasks

- [ ] Define new Go types: `Probe`, `HTTPProbe`, `FileProbe`, `CommandProbe`, `TCPProbe` and their assert structs in `internal/suite/suite.go`
- [ ] Add `Probes []Probe` field to `Correctness` struct alongside existing fields
- [ ] Implement `HTTPProbeVerifier` in `internal/verifier/` (replaces `StateVerifier` logic, adds `status` and future assertion types)
- [ ] Implement `FileProbeVerifier` in `internal/verifier/`
- [ ] Implement `CommandProbeVerifier` in `internal/verifier/`
- [ ] Implement `TCPProbeVerifier` in `internal/verifier/`
- [ ] Wire probes into `BuildPipeline` in `internal/verifier/pipeline.go`
- [ ] Add YAML unmarshaling for the discriminator-key pattern (custom `UnmarshalYAML` on `Probe`)
- [ ] Add validation: exactly one type key per probe entry, required fields per probe type
- [ ] Migrate `state` → `probes` with backward compat: loader converts old `state` entries to `http` probes (with deprecation warning)
- [ ] Update `examples/correctness/suite.yaml` to use `probes`
- [ ] Add tests for each probe verifier
- [ ] Add tests for the migration path (`state` → `probes`)
- [ ] Add validation tests (malformed probes, missing assert fields, multiple type keys)

## Acceptance Criteria

- `correctness.probes` with `http`, `file`, `command`, and `tcp` types all work end-to-end
- Each probe's `assert` block uses explicit, named checks (no implicit semantics)
- Old `state` suites still load and run correctly (converted to `http` probes internally)
- `suite.Load()` emits a deprecation warning when `state` is used
- All existing correctness checks (`agent_exits_ok`, `output`, `compiles`, `script`, `judge`) continue to work unchanged
- Pipeline runs probes after `script` and before `judge` in the verification order
- `examples/correctness/suite.yaml` uses the new `probes` syntax
- `TestLoad_Examples` passes
