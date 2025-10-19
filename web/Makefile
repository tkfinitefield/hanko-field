GO ?= go
BIN_DIR := $(CURDIR)/bin
MAIN_PKG := ./cmd/web
BINARY := $(BIN_DIR)/hanko-web
TAILWIND := $(CURDIR)/tools/tailwindcss

.PHONY: run dev build fmt test tidy css css-watch tools

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

# --- Assets (Tailwind standalone CLI) ---
css: tools
	@mkdir -p public/assets
	@$(TAILWIND) -c tailwind.config.js -i assets/css/input.css -o public/assets/app.css --minify

css-watch: tools
	@mkdir -p public/assets
	@$(TAILWIND) -c tailwind.config.js -i assets/css/input.css -o public/assets/app.css --watch

tools:
	@if [ ! -x "$(TAILWIND)" ]; then \
		echo "Missing tailwindcss standalone binary at $(TAILWIND)"; \
		echo "Download from https://github.com/tailwindlabs/tailwindcss/releases and place it there (chmod +x)."; \
		exit 1; \
	fi
