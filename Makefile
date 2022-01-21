GO=go

default: all

all: build test
.PHONY: all

vendor:
	$(GO) mod tidy && $(GO) mod vendor
.PHONY: vendor

build:
	$(GO) build -o bin/haiku
.PHONY: build

build-release:
	env GOOS=darwin GOARCH=amd64 $(GO) build -o bin/haiku_darwin_amd64
	env GOOS=darwin GOARCH=arm64 $(GO) build -o bin/haiku_darwin_arm64
	env GOOS=linux GOARCH=386 $(GO) build -o bin/haiku_linux_386
	env GOOS=linux GOARCH=amd64 $(GO) build -o bin/haiku_linux_amd64
	env GOOS=linux GOARCH=arm64 $(GO) build -o bin/haiku_linux_arm64
	env GOOS=windows GOARCH=386 $(GO) build -o bin/haiku_windows_386
	env GOOS=windows GOARCH=amd64 $(GO) build -o bin/haiku_windows_amd64
.PHONY: build-release

test:
	$(GO) test ./... -race -v
.PHONY: test

help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
.PHONY: help
