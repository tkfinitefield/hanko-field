SHELL := /bin/bash

.PHONY: app-format app-analyze app-custom-lint app-test app-coverage app-ci

app-format:
	cd app && dart format .

app-analyze:
	cd app && flutter analyze --no-fatal-warnings --no-fatal-infos

app-custom-lint:
	cd app && dart run custom_lint

app-test:
	cd app && flutter test

app-coverage:
	cd app && flutter test --coverage && dart run tool/check_coverage.dart --lcov=coverage/lcov.info --min=60

app-ci: app-format app-analyze app-custom-lint app-coverage

