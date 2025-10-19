GO ?= go
BIN_DIR := $(CURDIR)/bin
MAIN_PKG := ./cmd/web
BINARY := $(BIN_DIR)/hanko-web

.PHONY: run dev build fmt test tidy

run:
	DEV=1 $(GO) run $(MAIN_PKG)

dev:
	@command -v air >/dev/null 2>&1 && air -c .air.toml || (echo "air not installed; falling back to 'make run'" && $(MAKE) run)

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) $(MAIN_PKG)

fmt:
	@gofmt -w .

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

.DEFAULT_GOAL := build
