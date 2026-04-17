---
title: "Implement runner registry"
id: "01kpcvjkv"
status: pending
priority: critical
type: feature
effort: small
tags: ["backend", "registry"]
dependencies: ["01kpcvhw5"]
created: "2026-04-17"
---

# Implement runner registry

## Objective

Create an `internal/registry` package that maps runner names to factory functions, enabling dynamic runner construction from suite config. This decouples the executor from specific runner implementations.

## Tasks

- [ ] Create `internal/registry/registry.go` with `Registry` struct, `New()`, `Register(name, factory)`, and `Create(name, config) (Runner, error)` methods
- [ ] Define `Factory` type as `func(config map[string]any) (agentrunner.Runner, error)`
- [ ] Return a clear error from `Create` when a runner name is not registered
- [ ] Add unit tests for Register, Create (success), and Create (unknown name)

## Acceptance Criteria

- [ ] `Registry.Register` + `Registry.Create` round-trips successfully for a registered factory
- [ ] `Registry.Create` with an unregistered name returns an error containing the name
- [ ] Config map is passed through to the factory function
- [ ] Unit tests cover registration, successful creation, and unknown-name error
