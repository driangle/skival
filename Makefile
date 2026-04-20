.PHONY: build install lint vet test check-lite check

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

lint:
	golangci-lint run ./...

# ── Tests ────────────────────────────────────────────────────────────
test:
	go test ./...

# ── Validation ──────────────────────────────────────────────────────
validate-examples: build
	@fail=0; \
	for f in examples/*/suite.yaml; do \
		output=$$(./apps/cli/skival validate "$$f" 2>&1) || { echo "FAIL $$f"; echo "$$output"; fail=1; }; \
	done; \
	if [ $$fail -eq 0 ]; then echo "all example suites valid"; else exit 1; fi

# ── Composite targets ───────────────────────────────────────────────
check-lite: vet lint build validate-examples  ## Compile + lint + validate examples. No tests.

check: check-lite test      ## Full validation: compile, lint, tests.
