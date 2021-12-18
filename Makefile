GO=go

default: all

all: build test
.PHONY: all

vendor:
	$(GO) mod tidy && $(GO) mod vendor
.PHONY: vendor

build:
	$(GO) build -o haiku
.PHONY: build

test:
	$(GO) test ./... -race -v
.PHONY: test

help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
.PHONY: help
