# Flutter Flavors (dev / stg / prod)

This project supports three environments. Use Android product flavors and Dart `--dart-define` for config. iOS can use Dart defines now; native schemes can be added later.

## Android
- Flavors configured in `app/android/app/build.gradle.kts` under `productFlavors`:
  - `dev`: `applicationIdSuffix .dev`, `versionNameSuffix -dev`, app name “Hanko Field Dev”
  - `stg`: `applicationIdSuffix .stg`, `versionNameSuffix -stg`, app name “Hanko Field Staging”
  - `prod`: no suffix, app name “Hanko Field”
- App label uses `@string/app_name` (see `AndroidManifest.xml`).

Run
- Debug (dev): `flutter run --flavor dev --dart-define FLAVOR=dev -t lib/main.dart` (Android)
- Staging: `flutter run --flavor stg --dart-define FLAVOR=stg -t lib/main.dart`
- Release example: `flutter build apk --flavor prod --dart-define FLAVOR=prod`

Notes
- `baseUrl` and title are provided by `app/lib/core/app/app_flavor.dart` via Riverpod providers.
- You can add per-flavor resources under `app/android/app/src/<flavor>/res` to override icons, strings, etc.

## iOS
Option A (Dart defines only)
- Use: `flutter run --dart-define FLAVOR=dev` (no `--flavor` required). Dart obtains `FLAVOR` via `String.fromEnvironment`.

Option B (Native schemes)
- Create Xcode schemes/configurations for `Dev`, `Stg`, `Prod` and set distinct bundle IDs.
- Map schemes to Flutter flavors: `flutter run --flavor dev --dart-define FLAVOR=dev`.

## Config in Dart
- `app/lib/core/app/app_flavor.dart` exposes `appFlavorProvider` and `appConfigProvider` with per-environment `baseUrl`/`displayName`.

## Splash & Icons
- Android splash uses `app/src/main/res/drawable/launch_background.xml` (white background, can add centered image).
- App icons are present (default). To customize per flavor, add `mipmap` assets in `src/dev|stg|prod/res/`.
- iOS LaunchScreen storyboard and AppIcon set exist under `ios/Runner/Assets.xcassets`.

## Next Steps (optional)
- Add CI matrix to build `dev/stg/prod` artifacts.
- Add `flutter_native_splash` and `flutter_launcher_icons` later to automate branding (requires fetching packages).
