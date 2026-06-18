# Localized Release Runbook

This runbook is the operational checklist for adding future Stone Signature
languages, updating store metadata, running fastlane release lanes, and closing
post-release multilingual cleanup. It assumes the canonical language registry
is `config/languages.json` and that release work starts from the current
milestone order in `doc/multilingual-release-plan.md`.

## Language Addition Flow

Use one route code at a time unless the release owner explicitly approves a
batch.

1. Confirm the target route code already exists in `config/languages.json`.
2. Keep `release.enabled=false` while translation, layout, metadata, and
   release checks are incomplete.
3. Add or update localization files for the target route code:
   - Flutter ARB and generated localization inputs under `app/lib/l10n/`.
   - App long-form JSON under `app/assets/i18n/`.
   - API content under `api/content/i18n/`.
   - Web content under `web/content/i18n/`.
   - Store copy under `release/store_metadata/source/`.
4. Use intention sidecars for approved English, brand, legal, or placeholder
   holdouts.
5. Run the translation and registry checks:

```sh
make i18n-check
make i18n-stubs-check
make i18n-holdouts-check
make i18n-layout-qa-check
make i18n-flag-stages-check
```

Flag progression must stay staged:

1. `disabled`
2. `render_only`
3. `app_selectable`
4. `web_indexed`
5. `store_release_enabled`

Do not combine a translation content patch and a public flag promotion unless
the release owner explicitly asks for a combined release commit. The preferred
release shape is content first, then one registry stage transition with fresh
QA evidence.

## Store Metadata Update Flow

Store metadata source lives in `release/store_metadata/source/`. Generated
platform output lives in:

- `release/store_metadata/google_play`
- `release/store_metadata/app_store`

For each store-release candidate language:

1. Confirm `release.android_store_locale` and `release.ios_store_locale` in
   `config/languages.json`.
2. Add or update `release/store_metadata/source/<route_code>.json`.
3. Confirm title, subtitle, short description, full description, keywords,
   release notes, support URL, marketing URL, privacy URL, and screenshot
   captions.
4. Regenerate and verify platform metadata:

```sh
make store-metadata-check
make google-play-metadata
make google-play-metadata-check
make app-store-metadata
make app-store-metadata-check
make screenshot-metadata
make screenshot-metadata-check
```

5. Keep screenshots and generated metadata in sync before enabling
   `release.enabled=true`.
6. Run secret guardrails before any fastlane command that can contact a store:

```sh
make release-secret-guardrails-check
```

## fastlane Release Flow

Android lanes are defined in `app/android/fastlane/Fastfile`. Run them from
`app/android`.

```sh
cd app/android
BUNDLE_PATH=/tmp/hanko-field-android-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-android-bundle-config \
bundle exec fastlane android metadata
```

For Google Play internal testing, provide a local ignored service account JSON
or CI secret path through `SUPPLY_JSON_KEY`.

```sh
cd app/android
SUPPLY_JSON_KEY=/path/to/google-play-service-account.json \
BUNDLE_PATH=/tmp/hanko-field-android-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-android-bundle-config \
bundle exec fastlane android internal
```

Production Android upload additionally requires explicit release signoff.

```sh
cd app/android
SUPPLY_JSON_KEY=/path/to/google-play-service-account.json \
RELEASE_SIGNOFF_PATH=/path/to/checked-production-release-signoff.json \
RELEASE_SIGNOFF_CONFIRMATION="I confirm the Stone Signature production release" \
BUNDLE_PATH=/tmp/hanko-field-android-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-android-bundle-config \
bundle exec fastlane android production
```

iOS lanes are defined in `app/ios/fastlane/Fastfile`. Run them from `app/ios`.

```sh
cd app/ios
BUNDLE_PATH=/tmp/hanko-field-ios-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-ios-bundle-config \
bundle exec fastlane ios metadata
```

For TestFlight, provide App Store Connect credentials through
`APP_STORE_CONNECT_API_KEY_PATH`.

```sh
cd app/ios
APP_STORE_CONNECT_API_KEY_PATH=/path/to/app-store-connect-api-key.json \
BUNDLE_PATH=/tmp/hanko-field-ios-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-ios-bundle-config \
bundle exec fastlane ios testflight_upload
```

Production iOS upload additionally requires explicit release signoff.

```sh
cd app/ios
APP_STORE_CONNECT_API_KEY_PATH=/path/to/app-store-connect-api-key.json \
RELEASE_SIGNOFF_PATH=/path/to/checked-production-release-signoff.json \
RELEASE_SIGNOFF_CONFIRMATION="I confirm the Stone Signature production release" \
BUNDLE_PATH=/tmp/hanko-field-ios-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-ios-bundle-config \
bundle exec fastlane ios production
```

Before production lanes:

```sh
make android-fastlane-check
make ios-fastlane-check
make release-secret-guardrails-check
make i18n-ci
```

Never commit Google Play service account JSON, App Store Connect API key JSON,
`.p8` files, Android keystores, `key.properties`, provisioning profiles,
exported `.aab`, `.apk`, `.ipa`, or local fastlane reports.

## Post-Release Monitoring and Cleanup

After staged rollout begins, replace local zero-count reviews with real log,
support, and store-review exports. Do not commit production exports if they
contain private customer data.

Run:

```sh
make i18n-diagnostics-check
make i18n-support-triage-check
make i18n-translation-patches-check
make i18n-migration-cleanup-check
```

Monitoring must cover:

- unsupported locale requests
- fallback locale decisions
- missing content files or missing localized keys
- checkout locale and preferred locale
- malformed translation or catalog parse failures
- support feedback grouped by locale, platform, and screen
- high-priority translation fixes and owners

Close the release only after:

```sh
make i18n-ci
```

## Rollback

For a localized content issue:

1. Patch the affected translation content.
2. Run `make i18n-check`.
3. Run the relevant platform metadata check if store copy changed.

For a release-enabled language issue:

1. Set `release.enabled=false` in `config/languages.json`.
2. Regenerate store metadata if metadata availability changed.
3. Run `make i18n-flag-stages-check`.
4. Run `make i18n-ci`.
5. Use Google Play or App Store Connect rollback or previous-build promotion
   only when store-side release controls are needed.

For an app-selectable or web-indexed language issue:

1. Set the affected `app.selectable=false` or `web.indexed=false`.
2. Regenerate affected app, web, sitemap, and metadata outputs.
3. Run `make i18n-check` and the narrowest affected QA gate.

## Validation Command Set

Use this command set when preparing the next localized release:

```sh
make i18n-check
make store-metadata-check
make google-play-metadata-check
make app-store-metadata-check
make screenshot-metadata-check
make android-fastlane-check
make ios-fastlane-check
make release-secret-guardrails-check
make i18n-ci
```
