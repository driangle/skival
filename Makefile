.PHONY: build install lint lint-filesize vet test test-e2e check-lite check install-hooks

# ── Build ────────────────────────────────────────────────────────────
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X 'github.com/driangle/skival/apps/cli/cmd.commit=$(COMMIT)'

build:
	go build -ldflags "$(LDFLAGS)" -o apps/cli/skival ./apps/cli

install:
	go build -ldflags "$(LDFLAGS)" -o $(shell go env GOPATH)/bin/skival ./apps/cli

# ── Lint & compile checks ───────────────────────────────────────────
vet:
	go vet ./...

# golangci-lint is pinned as a `go tool` dependency in go.mod, so this runs
# the exact same version as CI and builds it with the repo's Go toolchain.
# No separate install step is needed.
lint:
	go tool golangci-lint run ./...

# Enforce per-file line limits (300 non-test / 500 test). See
# scripts/check-file-length.sh and README.md ("Code size limits").
lint-filesize:
	bash scripts/check-file-length.sh

# ── Tests ────────────────────────────────────────────────────────────
test:
	go test ./...

# Real-agent enforcement checks (build tag `e2e`). These invoke the actual
# `claude` CLI and consume API credits, so they are excluded from `make test`
# and `check-lite`. They skip automatically when `claude` is not on PATH.
test-e2e:
	go test -tags e2e -run E2E -count=1 ./internal/executor

# ── Validation ──────────────────────────────────────────────────────
validate-examples: build
	@fail=0; \
	for f in examples/*/suite.yaml; do \
		output=$$(./apps/cli/skival validate "$$f" 2>&1) || { echo "FAIL $$f"; echo "$$output"; fail=1; }; \
	done; \
	if [ $$fail -eq 0 ]; then echo "all example suites valid"; else exit 1; fi

# ── Git hooks ────────────────────────────────────────────────────────
# Point git at the tracked .githooks/ directory so the pre-commit hook
# (which runs `make check-lite`) is active. Run once per clone.
install-hooks:
	git config core.hooksPath .githooks
	@echo "pre-commit hook installed (core.hooksPath -> .githooks)"

# ── Composite targets ───────────────────────────────────────────────
check-lite: vet lint lint-filesize build validate-examples  ## Compile + lint + size limits + validate examples. No tests.

check: check-lite test      ## Full validation: compile, lint, tests.
