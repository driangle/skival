.PHONY: install
install:
	go build -o $(shell go env GOPATH)/bin/skival ./apps/cli
