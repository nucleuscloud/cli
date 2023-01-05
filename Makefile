GO=go

BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse HEAD)
VERSION ?= $(shell git describe --tags --abbrev=0 | tr -d '\n')

default: all

all: lint vet build test
.PHONY: all

vet:
	$(GO) vet ./...
.PHONY: vet

vendor:
	$(GO) mod tidy -compat=1.17 && $(GO) mod vendor
.PHONY: vendor

lint:
	golangci-lint run
.PHONY: lint

build:
	# Note the ldflags here are used for local builds only. To see the ldflags at release time, check the .goreleaser.yaml file
	$(GO) build -o bin/nucleus -ldflags="-X 'github.com/nucleuscloud/cli/internal/version.buildDate=${BUILD_DATE}' -X 'github.com/nucleuscloud/cli/internal/version.gitCommit=${GIT_COMMIT}' -X 'github.com/nucleuscloud/cli/internal/version.gitVersion=${VERSION}'"
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
