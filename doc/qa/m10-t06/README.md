# M10-T06 Production Release Signoff

Date: 2026-06-18 JST

Scope: add and verify the production release signoff gate for Android and iOS
fastlane production lanes, and record the release checklist state before staged
production execution.

## Result

PASS for the production signoff gate.

- Android production lane exists and requires `RELEASE_SIGNOFF_PATH`.
- Android production lane requires the exact `RELEASE_SIGNOFF_CONFIRMATION`
  phrase before uploading to the Google Play production track.
- iOS production lane exists and requires `RELEASE_SIGNOFF_PATH`.
- iOS production lane requires the exact `RELEASE_SIGNOFF_CONFIRMATION` phrase
  before uploading a production App Store Connect binary.
- The required confirmation phrase is:
  `I confirm the Stone Signature production release`.
- The local signoff record is
  `doc/qa/m10-t06/production-release-signoff.json`.
- Production lanes also parse the signoff JSON and require
  `approval.approved_for_m10_t07_execution=true`; the committed M10-T06 record
  intentionally keeps that value `false` until internal/TestFlight evidence is
  available.

No Google Play production upload, App Store Connect production upload, App Store
review submission, staged rollout, polling, streaming, SSE, or WebSocket
behavior was performed in this task.

## Checklist State

Validation gates:

- Android release candidate evidence is present from `M10-T01`.
- iOS release candidate evidence is present from `M10-T02`.
- Deep-link and payment-return evidence is present from `M10-T03`.
- i18n and release secret guardrails pass.
- Google Play internal upload evidence is still pending because `M10-T04` is
  blocked on Google Play credentials.
- TestFlight upload evidence is still pending because `M10-T05` is blocked on
  App Store Connect credentials.

Release readiness:

- Production fastlane lanes are guarded and ready for a future explicit
  signoff. They will reject the current M10-T06 signoff record until the
  release owner updates it for `M10-T07`.
- Production release execution remains blocked until `M10-T04` and `M10-T05`
  have real upload/install evidence and the release owner intentionally runs
  `M10-T07`.

## Validation

```sh
node --check scripts/release/android_fastlane_config.mjs
node --check scripts/release/ios_fastlane_config.mjs
make android-fastlane-check
make android-fastlane-test
make ios-fastlane-check
make ios-fastlane-test
make release-secret-guardrails-check
make release-secret-guardrails-test
jq empty doc/qa/m10-t06/production-release-signoff.json
make i18n-ci
git diff --check
git diff --cached --check
```

## Production Lane Inputs

Android production lane:

```sh
cd app/android
SUPPLY_JSON_KEY=/path/to/google-play-service-account.json \
RELEASE_SIGNOFF_PATH=/path/to/checked-production-release-signoff.json \
RELEASE_SIGNOFF_CONFIRMATION="I confirm the Stone Signature production release" \
BUNDLE_PATH=/tmp/hanko-field-android-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-android-bundle-config \
bundle exec fastlane android production
```

iOS production lane:

```sh
cd app/ios
APP_STORE_CONNECT_API_KEY_PATH=/path/to/app-store-connect-api-key.json \
RELEASE_SIGNOFF_PATH=/path/to/checked-production-release-signoff.json \
RELEASE_SIGNOFF_CONFIRMATION="I confirm the Stone Signature production release" \
BUNDLE_PATH=/tmp/hanko-field-ios-bundle \
BUNDLE_APP_CONFIG=/tmp/hanko-field-ios-bundle-config \
bundle exec fastlane ios production
```

Do not run either production lane until `M10-T07` and the release owner has
updated the signoff record for actual production approval.
