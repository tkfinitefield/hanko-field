# Release Secret Guardrails

This note defines where mobile release secrets may live and what must never be
committed.

## Allowed Secret Sources

- Local developer machines outside git.
- CI secret storage.
- Temporary files created by CI at runtime and removed after the job.

## Required Environment Variables

- `SUPPLY_JSON_KEY`: path to the Google Play service account JSON used by
  Android fastlane lanes that contact Google Play.
- `APP_STORE_CONNECT_API_KEY_PATH`: path to the App Store Connect API key JSON
  used by iOS fastlane lanes that contact App Store Connect or TestFlight.
- `SUPPLY_VALIDATE_ONLY`: optional Android override. Leave unset or `true`
  until the Google Play internal lane is intentionally allowed to upload.
- `TESTFLIGHT_SKIP_SUBMISSION`: optional iOS override. Leave unset or `true`
  until TestFlight submission behavior is intentionally changed.

## Files That Must Not Be Committed

- `app/android/key.properties`
- Android keystores, including `.jks` and `.keystore` files
- Google Play service account JSON files
- App Store Connect `AuthKey*.p8` files
- App Store Connect API key JSON files
- Apple provisioning profiles
- Exported `.aab`, `.apk`, and `.ipa` binaries
- Local fastlane `report.xml` files and generated fastlane `README.md` files

## Android Setup Notes

Gradle release signing reads `app/android/key.properties`, which should point
to a local keystore path. The local keystore and password material must stay out
of git.

Before hosting Android App Links association files, retrieve and validate the
release signing certificate SHA-256 fingerprint. The value must match the
certificate used for `org.finitefield.hankofield`.

## iOS Setup Notes

Do not commit Apple team IDs, App Store Connect API keys, `.p8` files, or
provisioning profiles. Provide App Store Connect credentials through
`APP_STORE_CONNECT_API_KEY_PATH` on a local machine or through CI secrets.

## Verification

Run:

```sh
make release-secret-guardrails-check
make release-secret-guardrails-test
make i18n-ci
```

These checks confirm that known release-secret paths are ignored and that
tracked files do not include signing secrets, store credentials, exported
release binaries, or local fastlane reports.
