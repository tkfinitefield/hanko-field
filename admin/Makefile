SHELL := /bin/bash

GOCACHE ?= $(PWD)/.gocache
BIN_DIR ?= $(PWD)/bin
TAILWIND_VERSION ?= 3.4.14
TAILWIND_INPUT := ./web/styles/tailwind.css
TAILWIND_OUTPUT := ./public/static/app.css

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
  TAILWIND_OS := macos
else ifeq ($(UNAME_S),Linux)
  TAILWIND_OS := linux
else
  $(error Unsupported OS $(UNAME_S). Please install Tailwind manually.)
endif

ifeq ($(UNAME_M),x86_64)
  TAILWIND_ARCH := x64
else ifeq ($(UNAME_M),amd64)
  TAILWIND_ARCH := x64
else ifeq ($(UNAME_M),arm64)
  TAILWIND_ARCH := arm64
else ifeq ($(UNAME_M),aarch64)
  TAILWIND_ARCH := arm64
else
  $(error Unsupported architecture $(UNAME_M). Please install Tailwind manually.)
endif

TAILWIND_BIN := $(BIN_DIR)/tailwindcss
TAILWIND_URL := https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)

.PHONY: dev
dev: templ tidy ensure-tailwind ## Start hot reload server with Tailwind watcher
	@set -euo pipefail; \
	echo "Starting Tailwind watcher..."; \
	$(TAILWIND_BIN) -i $(TAILWIND_INPUT) -o $(TAILWIND_OUTPUT) --watch & \
	TW_PID=$$!; \
	trap "kill $$TW_PID" EXIT; \
	echo "Starting air (Go hot reload)..."; \
	air

.PHONY: templ
templ: ## Regenerate templ components
	@mkdir -p $(GOCACHE)
	@GOCACHE=$(GOCACHE) go generate ./internal/admin/templates/...

.PHONY: css
css: ensure-tailwind ## Build Tailwind assets
	@$(TAILWIND_BIN) -i $(TAILWIND_INPUT) -o $(TAILWIND_OUTPUT) --minify

.PHONY: css-watch
css-watch: ensure-tailwind ## Watch Tailwind assets
	@$(TAILWIND_BIN) -i $(TAILWIND_INPUT) -o $(TAILWIND_OUTPUT) --watch

.PHONY: tidy
tidy: ## Run go mod tidy
	@mkdir -p $(GOCACHE)
	@GOCACHE=$(GOCACHE) go mod tidy

.PHONY: lint
lint: ## Run gofmt and go vet
	@gofmt -w $$(find . -name '*.go' -not -path "./.gocache/*")
	@mkdir -p $(GOCACHE)
	@GOCACHE=$(GOCACHE) go vet ./...

.PHONY: ensure-tailwind
ensure-tailwind: ## Download Tailwind standalone binary if missing
	@mkdir -p $(BIN_DIR)
	@if [ ! -x $(TAILWIND_BIN) ]; then \
		echo "Downloading tailwindcss $(TAILWIND_VERSION) ($(TAILWIND_OS)-$(TAILWIND_ARCH))..."; \
		curl -sSL -o $(TAILWIND_BIN) $(TAILWIND_URL); \
		chmod +x $(TAILWIND_BIN); \
	fi

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf $(GOCACHE) tmp
