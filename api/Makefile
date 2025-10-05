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
GOCOVMERGE_VERSION := latest

TOOLS := \
	golang.org/x/tools/cmd/gofumpt@$(GOFUMPT_VERSION) \
	github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) \
	honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) \
	golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) \
	github.com/wadey/gocovmerge@$(GOCOVMERGE_VERSION)

GO_SOURCES := $(shell find . -name '*.go' -not -path './bin/*' -not -path './vendor/*')
FIRESTORE_EMULATOR_HOST ?= localhost:8080
GOOGLE_CLOUD_PROJECT ?= hanko-field-dev

.PHONY: deps fmt fmt-check lint test test-integration test-ci cover run build clean generate vuln tidy docker-build docker-run

deps:
	@echo "==> Installing tools"
	@mkdir -p $(GOBIN)
	@for tool in $(TOOLS); do \
		GOBIN=$(GOBIN) $(GO) install $$tool; \
	done

tidy:
	@$(GO) mod tidy

fmt: deps
	@if [ -z "$(GO_SOURCES)" ]; then \
		echo "no Go source files found"; \
		exit 0; \
	fi
	@$(GOBIN)/gofumpt -w $(GO_SOURCES)

fmt-check: deps
	@if [ -z "$(GO_SOURCES)" ]; then \
		echo "no Go source files found"; \
		exit 0; \
	fi
	@CHANGED=$$($(GOBIN)/gofumpt -l $(GO_SOURCES)); \
	if [ -n "$$CHANGED" ]; then \
		echo "The following files require gofumpt formatting:"; \
		echo "$$CHANGED"; \
		exit 1; \
	fi

lint: deps
	@$(GOBIN)/golangci-lint run
	@$(GOBIN)/staticcheck ./...

test:
	@$(GO) test $(PACKAGES)

test-integration:
	@echo "==> Running integration tests with Firestore emulator at $(FIRESTORE_EMULATOR_HOST)"
	@FIRESTORE_EMULATOR_HOST=$(FIRESTORE_EMULATOR_HOST) \
		GOOGLE_CLOUD_PROJECT=$(GOOGLE_CLOUD_PROJECT) \
		$(GO) test -tags=integration -count=1 $(PACKAGES)

test-ci: test test-integration

cover: deps
	@$(GO) test $(PACKAGES) -covermode=atomic -coverprofile=coverage.unit.out
	@FIRESTORE_EMULATOR_HOST=$(FIRESTORE_EMULATOR_HOST) \
		GOOGLE_CLOUD_PROJECT=$(GOOGLE_CLOUD_PROJECT) \
		$(GO) test -tags=integration -count=1 $(PACKAGES) -covermode=atomic -coverprofile=coverage.integration.out
	@$(GOBIN)/gocovmerge coverage.unit.out coverage.integration.out > coverage.out

run:
	@$(GO) run $(MAIN_PKG)

build:
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BINARY) $(MAIN_PKG)

docker-build:
	@docker build --target runner -t hanko-field/api:local .

docker-run: docker-build
	@docker run --rm -e PORT=${PORT:-8080} -p ${PORT:-8080}:${PORT:-8080} hanko-field/api:local

clean:
	@rm -rf $(BIN_DIR) coverage.unit.out coverage.integration.out coverage.out

generate:
	@$(GO) generate $(PACKAGES)

vuln: deps
	@$(GOBIN)/govulncheck ./...

.DEFAULT_GOAL := build
