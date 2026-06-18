# Release Screenshot Workflow

This note defines deterministic screenshot slots for Google Play and App Store
metadata preparation. It does not require screenshot images to exist yet.

## Source Slots

Capture or copy source screenshots with this pattern:

```text
release/store_metadata/screenshots/source/{route_code}/{device}/{NN}-{key}.png
```

Allowed `device` values:

- `phone_6_5`
- `tablet_12_9`

Allowed screenshot keys:

- `design`
- `stones`
- `checkout`

The slot number must match the key order:

- `01-design.png`
- `02-stones.png`
- `03-checkout.png`

Captions come from
`release/store_metadata/source/{route_code}.json` under
`screenshot_captions`.

## Prepared Platform Paths

The generated manifest maps each source slot to platform preparation paths:

```text
release/store_metadata/screenshots/google_play/{android_store_locale}/{device}/{NN}-{key}.png
release/store_metadata/screenshots/app_store/{ios_store_locale}/{device}/{NN}-{key}.png
```

Keep this screenshot preparation tree separate from the fastlane metadata text
folders until screenshot upload is intentionally enabled. Current metadata lanes
skip screenshot upload.

## Commands

Generate the manifest:

```sh
make screenshot-metadata
```

Validate the manifest:

```sh
make screenshot-metadata-check
make screenshot-metadata-test
```

The manifest is generated from `config/languages.json` and
`release/store_metadata/source/*.json`, so locale names, store locales, and
captions stay aligned with the store metadata workflow.
