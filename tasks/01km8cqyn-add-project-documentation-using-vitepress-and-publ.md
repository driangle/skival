---
title: "Add project documentation using VitePress and publish to GitHub Pages"
id: "01km8cqyn"
status: completed
priority: medium
type: feature
tags: []
created: "2026-03-21"
completed_at: 2026-04-17
---

# Add project documentation using VitePress and publish to GitHub Pages

## Objective

Set up a VitePress-based documentation site for the skival project so users can browse guides, API references, and getting-started content. Automate deployment to GitHub Pages so docs stay up to date with every push to `main`.

## Tasks

- [x] Initialize a VitePress project (e.g. under `docs/`)
- [x] Configure VitePress theme, navigation, and sidebar structure
- [x] Write initial documentation pages (introduction, getting started, configuration, CLI usage)
- [x] Add a GitHub Actions workflow to build and deploy VitePress to GitHub Pages on push to `main`
- [x] Configure the repository's GitHub Pages settings to use the Actions deployment
- [x] Verify the published site is accessible and renders correctly

## Acceptance Criteria

- A `docs/` directory contains VitePress source files with at least an intro and getting-started page
- Running the VitePress dev server locally serves the documentation without errors
- A GitHub Actions workflow builds the docs and deploys to GitHub Pages on push to `main`
- The published documentation site is publicly accessible via the GitHub Pages URL
