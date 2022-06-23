GO=go

default: all

all: build test
.PHONY: all

vendor:
	$(GO) mod tidy -compat=1.17 && $(GO) mod vendor
.PHONY: vendor

build:
	golangci-lint run
	$(GO) build -o bin/nucleus
.PHONY: build

build-fast:
	$(GO) build -o bin/nucleus
.PHONY: build-fast

build-release:
	env GOOS=darwin GOARCH=amd64 $(GO) build -o bin/nucleus_darwin_amd64
	env GOOS=darwin GOARCH=arm64 $(GO) build -o bin/nucleus_darwin_arm64
	env GOOS=linux GOARCH=amd64 $(GO) build -o bin/nucleus_linux_amd64
	env GOOS=linux GOARCH=arm64 $(GO) build -o bin/nucleus_linux_arm64
	sha256sum bin/* >> bin/SHA256SUMS
.PHONY: build-release

test:
	$(GO) test ./... -race -v
.PHONY: test

test-fast:
	$(GO) test ./...
.PHONY: test-fast


help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
.PHONY: help
