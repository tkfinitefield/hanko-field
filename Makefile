SHELL := /bin/bash

.PHONY: app-format app-analyze app-custom-lint app-test app-coverage app-ci app-gen-l10n

app-format:
	cd app && dart format .

app-analyze:
	cd app && flutter analyze --no-fatal-warnings --no-fatal-infos

app-custom-lint:
	cd app && dart run custom_lint

app-gen-l10n:
	cd app && flutter gen-l10n

app-test:
	cd app && flutter test

app-coverage:
	cd app && flutter test --coverage && dart run tool/check_coverage.dart --lcov=coverage/lcov.info --min=60

app-ci: app-gen-l10n app-format app-analyze app-custom-lint app-coverage
