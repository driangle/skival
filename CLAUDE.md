Every new change or feature must include appropriate tests.
The CLI must not make assumptions about the user's projects or tech stack — it should rely on the user providing relevant information through the suite.yaml, CLI args, or other config files.

Enforced code size limits (checked by `make check-lite`): non-test files ≤ 300 lines, `_test.go` files ≤ 500 lines, functions ≤ 40 lines / ≤ 25 statements. Split oversized files into cohesive same-package files rather than growing "god files"; extract helpers to keep functions small. Only exempt a specific function with `//nolint:funlen // <reason>` when a split is genuinely not worthwhile. See README.md → Development for details and `make install-hooks`.
