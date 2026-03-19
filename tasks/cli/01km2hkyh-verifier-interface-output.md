---
title: "Verifier interface and output substring verifier"
id: "01km2hkyh"
status: completed
priority: medium
type: feature
tags: ["phase-2", "verifier"]
dependencies: ["01km2hkxn"]
created: "2026-03-19"
phase: phase-2
---

# Verifier interface and output substring verifier

## Objective

Define the Verifier interface and implement the first concrete verifier: output substring matching against expected_output strings in the eval's correctness config.

## Tasks

- [x] Define Verifier interface in `internal/verifier/verifier.go` with VerifyInput/VerifyResult types
- [x] Implement OutputVerifier in `internal/verifier/output.go` that checks RunOutput.Text contains all expected_output substrings
- [x] Return clear reasons on failure (which substring was missing)
- [x] Unit tests for match, partial match, no match, empty expected_output cases

## Acceptance Criteria

- Verifier interface is generic and not tied to any specific verification strategy
- OutputVerifier passes when all expected_output substrings are found in the run output
- OutputVerifier fails with a descriptive reason when any substring is missing
- Empty expected_output list is treated as a pass (no assertions to check)
