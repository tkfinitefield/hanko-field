GO ?= go
GOBIN ?= $(CURDIR)/bin
BIN_DIR := $(CURDIR)/bin
PACKAGES := ./...
MAIN_PKG := ./cmd/api
BINARY := $(BIN_DIR)/hanko-api

GOFUMPT_VERSION := v0.6.0
GOLANGCI_LINT_VERSION := v1.57.2
STATICCHECK_VERSION := 2024.1.2
GOVULNCHECK_VERSION := latest

TOOLS := \
	golang.org/x/tools/cmd/gofumpt@$(GOFUMPT_VERSION) \
	github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) \
	honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) \
	golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: deps fmt lint test run build clean generate vuln tidy

deps:
	@echo "==> Installing tools"
	@mkdir -p $(GOBIN)
	@for tool in $(TOOLS); do \
		GOBIN=$(GOBIN) $(GO) install $$tool; \
	done

tidy:
	@$(GO) mod tidy

fmt: deps
	@$(GOBIN)/gofumpt -w $$(find . -name '*.go' -not -path './bin/*' -not -path './vendor/*')

lint: deps
	@$(GOBIN)/golangci-lint run
	@$(GOBIN)/staticcheck ./...

test:
	@$(GO) test $(PACKAGES)

run:
	@$(GO) run $(MAIN_PKG)

build:
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BINARY) $(MAIN_PKG)

clean:
	@rm -rf $(BIN_DIR)

generate:
	@$(GO) generate $(PACKAGES)

vuln: deps
	@$(GOBIN)/govulncheck ./...

.DEFAULT_GOAL := build
