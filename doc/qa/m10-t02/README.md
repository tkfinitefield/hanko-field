# M10-T02 iOS Release Candidate

Date: 2026-06-18 JST

Scope: build and verify the signed iOS release candidate IPA for the current
multilingual release baseline.

## Result

PASS.

- Artifact: `app/build/ios/ipa/STONE SIGNATURE.ipa`
- Size: 31,355,322 bytes, reported by Flutter as 31.5MB
- SHA-256:
  `b74de0d4c300acfbdfba44f1c6423693835b999a2ea876b835cca1684349c427`
- Version: `1.1.0+11`
- Bundle identifier: `org.finitefield.hankofield`
- Team ID: `5267S9U4PR`
- Signer: `Apple Distribution: Finite Field, K.K. (5267S9U4PR)`
- Provisioning profile:
  `iOS Team Store Provisioning Profile: org.finitefield.hankofield`

The IPA is a local ignored build artifact and is not committed.

## Validation

```sh
cd app
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer flutter doctor -v
flutter build ipa --release --export-options-plist=ios/ExportOptions.plist
shasum -a 256 "build/ios/ipa/STONE SIGNATURE.ipa"
codesign --verify --deep --strict --verbose=2 Payload/Runner.app
codesign -dv --verbose=4 Payload/Runner.app
security cms -D -i Payload/Runner.app/embedded.mobileprovision
```

The local shell used a temporary `xcrun` wrapper because this Mac's active
developer directory is `/Library/Developer/CommandLineTools`; `DEVELOPER_DIR`
points the build at `/Applications/Xcode.app/Contents/Developer`. No local
absolute Xcode path is committed to the project.

## Build Notes

- `app/.metadata` now records the existing iOS platform, so Flutter recognizes
  the checked-in iOS project.
- `app/ios/Runner.xcodeproj/project.pbxproj` now records automatic signing for
  team `5267S9U4PR`.
- `app/ios/ExportOptions.plist` keeps App Store export settings in source and
  disables automatic build-number mutation during export.
- `app/ios/fastlane/Fastfile` uses the same export options for the future
  TestFlight lane.
- Flutter/Xcode added Swift Package Manager integration for the iOS plugin
  build path. The project still has CocoaPods integration, and Flutter reports
  that CocoaPods can be removed later to improve iOS build time.

## Private Material

The local IPA, App Store Connect keys, and provisioning profiles remain ignored:

- `app/build/ios/ipa/STONE SIGNATURE.ipa`
- `app/ios/AuthKey*.p8`
- `app/ios/fastlane/*api-key*.json`
- `app/ios/**/*.mobileprovision`

No IPA artifact, App Store Connect credential, provisioning profile, polling,
streaming, SSE, or WebSocket behavior is committed by this task.
