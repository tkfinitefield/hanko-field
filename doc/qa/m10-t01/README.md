# M10-T01 Android Release Candidate

Date: 2026-06-18 JST

Scope: build and verify the Android release candidate app bundle for the
current multilingual release baseline.

## Result

PASS.

- Artifact: `app/build/app/outputs/bundle/release/app-release.aab`
- Size: 62,753,384 bytes, reported by Flutter as 62.8MB
- SHA-256:
  `2ef6ff7862ce0a33731e4be1aaa92d19a2e4dd12bb8262b5f9fc853bb9f9f77c`
- Version: `1.1.0+11`
- Signer: `CN=Hanko Field, OU=Mobile, O=Hanko Field, L=Tokyo, ST=Tokyo, C=JP`
- Signature algorithm: SHA256withRSA, 2048-bit key

The AAB is a local ignored build artifact and is not committed.

## Validation

```sh
cd app && flutter build appbundle --release
shasum -a 256 app/build/app/outputs/bundle/release/app-release.aab
jarsigner -verify -verbose -certs app/build/app/outputs/bundle/release/app-release.aab
cd app && flutter test test/generated_hanko_localizations_test.dart
cd app && flutter test test/language_registry_test.dart
```

`jarsigner -verify -verbose -certs` reports `jar verified`. `jarsigner
-verify -strict` exits with code 4 because the upload certificate is self-signed
and the signature has no timestamp; this is recorded in
`android-release-candidate.json`.

## Build Notes

- Flutter added `android.builtInKotlin=false` and `android.newDsl=false` to
  `app/android/gradle.properties` during the release build migration check.
- Flutter regenerated `app/lib/l10n/generated/` for the 68-language ARB set
  created in M9-T01.
- The app runtime still uses `hankoSupportedLocales`, so selectable runtime
  app locales remain `en`, `ja`, `zh-Hans`, `zh-Hant`, and `ar`.
- Flutter warns that the app and `url_launcher_android` still use the Kotlin
  Gradle Plugin path that must migrate before a future Flutter version requires
  Built-in Kotlin.
- Flutter reports untranslated messages for disabled/stub locales. These are
  expected for the current M9 stub baseline and are blocked from release by the
  M9-T02, M9-T05, and M9-T06 gates.

## Private Material

The local release signing files are present and ignored:

- `app/android/key.properties`
- `app/android/app/upload-keystore.jks`

No signing material, AAB/APK artifact, upload credential, polling, streaming,
SSE, or WebSocket behavior is committed by this task.
