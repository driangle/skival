---
title: "Script and state assertion verifiers"
id: "01km2hkzt"
status: completed
priority: medium
type: feature
tags: ["phase-2", "verifier"]
dependencies: ["01km2hkyh"]
created: "2026-03-19"
phase: phase-2
---

# Script and state assertion verifiers

## Objective

Implement two additional verifiers: ScriptVerifier (runs a user-provided shell script, exit 0 = pass) and StateVerifier (makes HTTP requests and checks response bodies for expected strings).

## Tasks

- [x] Implement ScriptVerifier in `internal/verifier/script.go` — runs correctness.script as a shell command, passes run output via stdin or env, exit 0 = pass
- [x] Implement StateVerifier in `internal/verifier/state.go` — iterates correctness.state assertions, makes HTTP requests, checks response body contains expect string
- [x] Handle timeouts for both script execution and HTTP requests
- [x] Unit tests with mock script and mock HTTP server

## Acceptance Criteria

- ScriptVerifier executes the script in the eval's working directory and reports pass/fail based on exit code
- ScriptVerifier captures stderr as the failure reason
- StateVerifier makes GET/POST requests and checks response bodies
- Both verifiers respect context cancellation and timeouts
