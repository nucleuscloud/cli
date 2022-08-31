GO=go

default: all

all: lint build test
.PHONY: all

vendor:
	$(GO) mod tidy -compat=1.17 && $(GO) mod vendor
.PHONY: vendor

lint:
	golangci-lint run
.PHONY: lint

build:
	$(GO) build -o bin/nucleus
.PHONY: build

test:
	$(GO) test ./... -race -v
.PHONY: test

test-fast:
	$(GO) test ./...
.PHONY: test-fast

help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
.PHONY: help
