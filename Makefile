.PHONY: build install lint vet test check-lite check

# ── Build ────────────────────────────────────────────────────────────
build:
	go build -o apps/cli/skival ./apps/cli

install:
	go build -o $(shell go env GOPATH)/bin/skival ./apps/cli

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
	@for f in examples/*/suite.yaml; do \
		echo "validating $$f"; \
		./apps/cli/skival validate "$$f" || exit 1; \
	done

# ── Composite targets ───────────────────────────────────────────────
check-lite: vet lint build validate-examples  ## Compile + lint + validate examples. No tests.

check: check-lite test      ## Full validation: compile, lint, tests.
