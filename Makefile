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

# ── Composite targets ───────────────────────────────────────────────
check-lite: vet lint build  ## Compile + lint. No tests.

check: check-lite test      ## Full validation: compile, lint, tests.
