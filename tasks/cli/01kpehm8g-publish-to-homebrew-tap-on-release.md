---
title: "Publish to Homebrew tap on release"
id: "01kpehm8g"
status: completed
priority: medium
type: feature
tags: ["release", "homebrew", "ci"]
created: "2026-04-17"
completed_at: 2026-04-17
---

# Publish to Homebrew tap on release

## Objective

Set up the full release pipeline so that tagging a version (e.g. `v0.1.0`) builds cross-platform binaries, creates a GitHub release, and publishes a Homebrew formula to [driangle/homebrew-tap](https://github.com/driangle/homebrew-tap).

## Tasks

- [x] Add a version constant to the Go CLI (e.g. `apps/cli/cmd/root.go`) that can be set via `-ldflags` at build time
- [x] Create `.release.conf` listing version files for the `/release` skill
- [x] Create `.github/workflows/release.yml` triggered on `v*` tags that:
  - Runs CI checks (vet, lint, test)
  - Builds binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
  - Creates a GitHub release with the binaries using `softprops/action-gh-release`
- [x] Add a Homebrew formula update step to the release workflow that pushes to `driangle/homebrew-tap`
- [x] Verify end-to-end: dry-run `/release` and inspect the generated workflow artifacts

## Acceptance Criteria

- `.release.conf` exists and is compatible with the `/release` skill
- `.github/workflows/release.yml` triggers on `v*` tags and builds multi-platform binaries
- GitHub release is created automatically with attached binaries and release notes
- Homebrew formula in `driangle/homebrew-tap` is updated on each release
- `taskmd validate` passes
