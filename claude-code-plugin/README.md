# Skival Claude Code Plugin

A Claude Code plugin that adds a `/skival` skill for generating eval suite.yaml files to benchmark AI agent skills.

## Installation

### From marketplace

```bash
# Add the marketplace (from this repo)
/plugin marketplace add <repo-url>

# Install the plugin
/plugin install skival@skival-marketplace
```

### Local development

```bash
# Load directly from the plugin directory
claude --plugin-dir ./claude-code-plugin
```

## Usage

Once installed, use the `/skival` skill in Claude Code:

```
/skival Create a suite that compares Sonnet vs Opus on a fibonacci task
```

The skill will generate a valid `suite.yaml` with appropriate eval structure, correctness checks, and treatment configurations.

## What's included

- **`/skival` skill** - Generates suite.yaml files with full knowledge of the skival schema, validation rules, common patterns, and best practices.

## Structure

```
skival/                              # Repo root
├── .claude-plugin/
│   └── marketplace.json             # Marketplace catalog (repo-level)
└── claude-code-plugin/
    ├── .claude-plugin/
    │   └── plugin.json              # Plugin manifest
    ├── skills/
    │   └── skival/
    │       └── SKILL.md             # The /skival skill definition
    └── README.md
```
