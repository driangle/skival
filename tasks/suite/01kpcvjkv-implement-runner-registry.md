---
title: "Implement runner registry"
id: "01kpcvjkv"
status: completed
priority: critical
type: feature
effort: small
tags: ["backend", "registry"]
dependencies: ["01kpcvhw5"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Implement runner registry

## Objective

Create an `internal/registry` package that maps runner names to factory functions, enabling dynamic runner construction from suite config. This decouples the executor from specific runner implementations.

## Tasks

- [x] Create `internal/registry/registry.go` with `Registry` struct, `New()`, `Register(name, factory)`, and `Create(name, config) (Runner, error)` methods
- [x] Define `Factory` type as `func(config map[string]any) (agentrunner.Runner, error)`
- [x] Return a clear error from `Create` when a runner name is not registered
- [x] Add unit tests for Register, Create (success), and Create (unknown name)

## Acceptance Criteria

- [x] `Registry.Register` + `Registry.Create` round-trips successfully for a registered factory
- [x] `Registry.Create` with an unregistered name returns an error containing the name
- [x] Config map is passed through to the factory function
- [x] Unit tests cover registration, successful creation, and unknown-name error
