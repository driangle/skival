---
title: "CI code coverage reporting"
id: "01kpfdxsn"
status: pending
priority: low
type: chore
tags: ["ci", "testing"]
created: "2026-04-18"
---

# CI code coverage reporting

## Description

The CI pipeline runs `go test ./...` but doesn't track or report code coverage. Adding coverage reporting enables tracking coverage trends over time and identifying untested areas.

## Tasks

- [ ] Update `ci.yaml` to run tests with `-coverprofile=coverage.out` and `-covermode=atomic`
- [ ] Add a step to upload coverage to Codecov (or similar service)
- [ ] Add a coverage badge to README.md
- [ ] Optionally add a minimum coverage threshold that fails CI if coverage drops below it
