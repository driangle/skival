---
title: "Suite YAML types and loader"
id: "01km2hkvf"
status: pending
priority: high
type: feature
tags: ["phase-1", "config"]
dependencies: ["01km2hk6e"]
created: "2026-03-19"
phase: phase-1
---

# Suite YAML types and loader

## Objective

Define Go types for the suite configuration schema (Suite, Eval, Treatment, Correctness, Setup) and implement YAML loading with defaults merging and validation.

## Tasks

- [ ] Define types in `internal/suite/suite.go`: Suite, Eval, Treatment, Correctness, Setup, StateAssertion
- [ ] Implement YAML loader in `internal/suite/loader.go` that reads a suite file and unmarshals into Suite struct
- [ ] Implement defaults merging — suite-level defaults (samples, timeout, model) applied to evals unless overridden
- [ ] Implement validation in `internal/suite/validate.go`: required fields, unique eval IDs, at least one treatment per eval, valid complexity values
- [ ] Write unit tests for loading, defaults merging, and validation errors

## Acceptance Criteria

- Can load the example suite.yaml from PLAN.md and get a fully populated Suite struct
- Suite-level defaults are correctly inherited by evals that don't override them
- Validation returns clear errors for missing required fields, duplicate IDs, invalid values
- Unit tests cover happy path and error cases
