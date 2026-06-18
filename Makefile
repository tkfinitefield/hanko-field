COMPOSE ?= docker compose
WORKSPACE_SERVICE ?= workspace

ENV ?= dev
ENV_FILE ?= .env.$(ENV)

API_PORT ?= 3050
ADMIN_PORT ?= 3051
WEB_PORT ?= 3052
MODE ?= mock
LOCALE ?= ja
STRIPE_WEBHOOK_URL ?= http://localhost:3050/v1/payments/stripe/webhook
GCP_PROD_PROJECT ?= hanko-field-prod
GCP_PROD_REGION ?= asia-northeast1

ADMIN_MODE_EXPORT := $(if $(HANKO_ADMIN_MODE),export HANKO_ADMIN_MODE=$(HANKO_ADMIN_MODE);,)
WEB_MODE_EXPORT := $(if $(HANKO_WEB_MODE),export HANKO_WEB_MODE=$(HANKO_WEB_MODE);,)

.PHONY: help docker-up docker-down docker-shell docker-api docker-admin docker-web docker-dev stripe-listen deploy-web-prod store-metadata-check store-metadata-test google-play-metadata google-play-metadata-check google-play-metadata-test app-store-metadata app-store-metadata-check app-store-metadata-test screenshot-metadata screenshot-metadata-check screenshot-metadata-test android-fastlane-check android-fastlane-test ios-fastlane-check ios-fastlane-test release-secret-guardrails-check release-secret-guardrails-test i18n-registry-test i18n-status i18n-status-test i18n-todo i18n-todo-test i18n-check i18n-check-test i18n-arb-test i18n-json-shape-test i18n-intentions-test i18n-export i18n-import i18n-handoff-test i18n-ci

ifneq ($(wildcard $(ENV_FILE)),)
COMPOSE_ENV_FILE_OPT := --env-file $(ENV_FILE)
else
COMPOSE_ENV_FILE_OPT :=
endif

ENV_LOAD_CMD = HANKO_ADMIN_MODE_SAVED=$${HANKO_ADMIN_MODE:-}; HANKO_WEB_MODE_SAVED=$${HANKO_WEB_MODE:-}; if [ -f "/workspace/$(ENV_FILE)" ]; then set -a; . "/workspace/$(ENV_FILE)"; set +a; fi; if [ -n "$$HANKO_ADMIN_MODE_SAVED" ]; then export HANKO_ADMIN_MODE="$$HANKO_ADMIN_MODE_SAVED"; fi; if [ -n "$$HANKO_WEB_MODE_SAVED" ]; then export HANKO_WEB_MODE="$$HANKO_WEB_MODE_SAVED"; fi;

help:
	@echo "Available targets:"
	@echo "  make docker-up      # Build and start workspace container"
	@echo "  make docker-down    # Stop containers"
	@echo "  make docker-shell   # Open devbox shell in workspace container"
	@echo "  make docker-api     # Run API server in container"
	@echo "  make docker-admin   # Run Admin server in container"
	@echo "  make docker-web     # Run Web server in container"
	@echo "  make docker-dev     # Run API/Admin/Web together in container"
	@echo "  make deploy-web-prod # Deploy Web to Cloud Run with explicit project/region"
	@echo "  make stripe-listen  # Forward Stripe webhooks to the local API"
	@echo "  make store-metadata-check # Validate release store metadata source JSON"
	@echo "  make store-metadata-test # Validate release store metadata helpers"
	@echo "  make google-play-metadata # Generate Google Play metadata folders"
	@echo "  make google-play-metadata-check # Check generated Google Play metadata"
	@echo "  make google-play-metadata-test # Validate Google Play metadata generator"
	@echo "  make app-store-metadata # Generate App Store metadata folders"
	@echo "  make app-store-metadata-check # Check generated App Store metadata"
	@echo "  make app-store-metadata-test # Validate App Store metadata generator"
	@echo "  make screenshot-metadata # Generate store screenshot manifest"
	@echo "  make screenshot-metadata-check # Check generated store screenshot manifest"
	@echo "  make screenshot-metadata-test # Validate store screenshot workflow"
	@echo "  make android-fastlane-check # Validate Android fastlane metadata lanes"
	@echo "  make android-fastlane-test # Test Android fastlane config checks"
	@echo "  make ios-fastlane-check # Validate iOS fastlane metadata lanes"
	@echo "  make ios-fastlane-test # Test iOS fastlane config checks"
	@echo "  make release-secret-guardrails-check # Validate release secret ignore rules"
	@echo "  make release-secret-guardrails-test # Test release secret guardrails"
	@echo "  make i18n-status    # Report localization registry and missing files"
	@echo "  make i18n-todo      # Report missing localization keys"
	@echo "  make i18n-check     # Validate localization registry, files, and missing keys"
	@echo "  make i18n-arb-test  # Validate ARB placeholder and ICU checks"
	@echo "  make i18n-json-shape-test # Validate JSON shape and fallback checks"
	@echo "  make i18n-intentions-test # Validate intention sidecar checks"
	@echo "  make i18n-export    # Export translation handoff JSON"
	@echo "  make i18n-import    # Import translation handoff JSON with IN=<file>"
	@echo "  make i18n-handoff-test # Validate translation handoff helpers"
	@echo "  make i18n-ci        # Run localization checks used by CI"
	@echo "  make i18n-registry-test # Validate the language registry parser"
	@echo ""
	@echo "Options:"
	@echo "  ENV=dev|prod"
	@echo "  ENV_FILE=.env.dev|.env.prod"
	@echo "  STRIPE_WEBHOOK_URL=http://localhost:3050/v1/payments/stripe/webhook"

docker-up:
	$(COMPOSE) $(COMPOSE_ENV_FILE_OPT) build
	$(COMPOSE) $(COMPOSE_ENV_FILE_OPT) up -d $(WORKSPACE_SERVICE)

docker-down:
	$(COMPOSE) $(COMPOSE_ENV_FILE_OPT) down

docker-shell:
	$(COMPOSE) exec $(WORKSPACE_SERVICE) sh -lc '$(ENV_LOAD_CMD) exec devbox shell'

docker-api:
	$(COMPOSE) exec $(WORKSPACE_SERVICE) sh -lc 'set -e; $(ENV_LOAD_CMD) cd /workspace && devbox run -- make -C api run PORT=$${API_SERVER_PORT:-$(API_PORT)}'

docker-admin:
	$(COMPOSE) exec $(WORKSPACE_SERVICE) sh -lc 'set -e; $(ADMIN_MODE_EXPORT) $(ENV_LOAD_CMD) cd /workspace && devbox run -- make -C admin dev PORT=$${ADMIN_PORT:-$(ADMIN_PORT)} MODE=$${HANKO_ADMIN_MODE:-$(MODE)} LOCALE=$${HANKO_ADMIN_LOCALE:-$(LOCALE)}'

docker-web:
	$(COMPOSE) exec $(WORKSPACE_SERVICE) sh -lc 'set -e; $(WEB_MODE_EXPORT) $(ENV_LOAD_CMD) cd /workspace && devbox run -- make -C web dev PORT=$${HANKO_WEB_PORT:-$(WEB_PORT)} MODE=$${HANKO_WEB_MODE:-$(MODE)} LOCALE=$${HANKO_WEB_LOCALE:-$(LOCALE)}'

docker-dev:
	$(COMPOSE) exec $(WORKSPACE_SERVICE) sh -lc 'set -e; $(ADMIN_MODE_EXPORT) $(WEB_MODE_EXPORT) $(ENV_LOAD_CMD) cd /workspace; exec devbox run -- bash ./scripts/docker-dev.sh'

deploy-web-prod:
	./scripts/deploy-web-prod.sh

stripe-listen:
	@set -e; if [ -f "$(ENV_FILE)" ]; then set -a; . "$(ENV_FILE)"; set +a; fi; stripe listen --forward-to "$${STRIPE_WEBHOOK_URL:-$(STRIPE_WEBHOOK_URL)}"

i18n-registry-test:
	node --test scripts/i18n/registry.test.mjs

i18n-status:
	node scripts/i18n/status.mjs

i18n-status-test:
	node --test scripts/i18n/status.test.mjs

i18n-todo:
	node scripts/i18n/todo.mjs

i18n-todo-test:
	node --test scripts/i18n/todo.test.mjs

i18n-check:
	node scripts/i18n/check.mjs

i18n-check-test:
	node --test scripts/i18n/check.test.mjs

i18n-arb-test:
	node --test scripts/i18n/arb.test.mjs

i18n-json-shape-test:
	node --test scripts/i18n/json_shape.test.mjs

i18n-intentions-test:
	node --test scripts/i18n/intentions.test.mjs

i18n-export:
	@node scripts/i18n/handoff.mjs export

i18n-import:
	@node scripts/i18n/handoff.mjs import

i18n-handoff-test:
	node --test scripts/i18n/handoff.test.mjs

store-metadata-check:
	node scripts/release/store_metadata.mjs

store-metadata-test:
	node --test scripts/release/store_metadata.test.mjs

google-play-metadata:
	node scripts/release/google_play_metadata.mjs

google-play-metadata-check:
	node scripts/release/google_play_metadata.mjs --check

google-play-metadata-test:
	node --test scripts/release/google_play_metadata.test.mjs

app-store-metadata:
	node scripts/release/app_store_metadata.mjs

app-store-metadata-check:
	node scripts/release/app_store_metadata.mjs --check

app-store-metadata-test:
	node --test scripts/release/app_store_metadata.test.mjs

screenshot-metadata:
	node scripts/release/screenshot_metadata.mjs

screenshot-metadata-check:
	node scripts/release/screenshot_metadata.mjs --check

screenshot-metadata-test:
	node --test scripts/release/screenshot_metadata.test.mjs

android-fastlane-check:
	node scripts/release/android_fastlane_config.mjs

android-fastlane-test:
	node --test scripts/release/android_fastlane_config.test.mjs

ios-fastlane-check:
	node scripts/release/ios_fastlane_config.mjs

ios-fastlane-test:
	node --test scripts/release/ios_fastlane_config.test.mjs

release-secret-guardrails-check:
	node scripts/release/secret_guardrails.mjs

release-secret-guardrails-test:
	node --test scripts/release/secret_guardrails.test.mjs

i18n-ci:
	node --check scripts/i18n/registry.mjs
	node --check scripts/i18n/status.mjs
	node --check scripts/i18n/todo.mjs
	node --check scripts/i18n/check.mjs
	node --check scripts/i18n/arb.mjs
	node --check scripts/i18n/json_shape.mjs
	node --check scripts/i18n/intentions.mjs
	node --check scripts/i18n/handoff.mjs
	node --check scripts/release/store_metadata.mjs
	node --check scripts/release/google_play_metadata.mjs
	node --check scripts/release/app_store_metadata.mjs
	node --check scripts/release/screenshot_metadata.mjs
	node --check scripts/release/android_fastlane_config.mjs
	node --check scripts/release/ios_fastlane_config.mjs
	node --check scripts/release/secret_guardrails.mjs
	$(MAKE) i18n-check
	$(MAKE) store-metadata-check
	$(MAKE) google-play-metadata-check
	$(MAKE) app-store-metadata-check
	$(MAKE) screenshot-metadata-check
	$(MAKE) android-fastlane-check
	$(MAKE) ios-fastlane-check
	$(MAKE) release-secret-guardrails-check
	$(MAKE) i18n-check-test
	$(MAKE) i18n-arb-test
	$(MAKE) i18n-json-shape-test
	$(MAKE) i18n-intentions-test
	$(MAKE) i18n-handoff-test
	$(MAKE) i18n-todo-test
	$(MAKE) i18n-status-test
	$(MAKE) i18n-registry-test
	$(MAKE) store-metadata-test
	$(MAKE) google-play-metadata-test
	$(MAKE) app-store-metadata-test
	$(MAKE) screenshot-metadata-test
	$(MAKE) android-fastlane-test
	$(MAKE) ios-fastlane-test
	$(MAKE) release-secret-guardrails-test
