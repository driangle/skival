---
title: "Setup lifecycle hooks"
id: "01km2hm7g"
status: pending
priority: low
type: feature
tags: ["phase-4", "core"]
dependencies: ["01km2hkxn"]
created: "2026-03-19"
phase: phase-4
---

# Setup lifecycle hooks

## Objective

Implement the setup lifecycle hooks (before, after, reset) so evals can manage external dependencies like databases, backends, or Docker services during the eval run.

## Tasks

- [ ] Execute setup.before shell command before the first treatment of an eval
- [ ] Execute setup.reset between treatments (not before the first one)
- [ ] Execute setup.after after all treatments complete (even on error)
- [ ] Run commands in the eval's dir with the eval's env
- [ ] Capture and log stdout/stderr from hook commands
- [ ] Fail the eval if before/reset hooks fail; after hook failures are warnings only

## Acceptance Criteria

- `setup.before` runs once before any treatment starts
- `setup.reset` runs between each treatment to restore clean state
- `setup.after` always runs, even if treatments failed
- Hook failures in before/reset mark the eval as errored
- Hook output is visible in verbose/debug mode
