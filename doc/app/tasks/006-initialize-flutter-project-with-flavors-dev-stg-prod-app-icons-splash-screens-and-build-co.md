# Initialize Flutter project with flavors (dev/stg/prod), app icons, splash screens, and build configurations.

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 006

## Goal
Bootstrap Flutter workspace with flavors and branding assets.

## Implementation Steps
1. Run `flutter create` with appropriate organization ID and null safety enabled.
2. Configure flavors (dev/stg/prod) via separate entrypoints and native scheme settings.
3. Implement splash screens and adaptive icons using `flutter_native_splash` and `flutter_launcher_icons`.
4. Set bundle identifiers/package names per flavor and configure build configs.
5. Document commands for running each flavor locally and in CI.
