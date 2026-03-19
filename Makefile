.PHONY: build install

build:
	go build -o apps/cli/skival ./apps/cli

install:
	go build -o $(shell go env GOPATH)/bin/skival ./apps/cli
