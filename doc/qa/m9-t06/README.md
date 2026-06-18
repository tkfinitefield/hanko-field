# M9-T06 Release-Candidate Translation Freeze

Date: 2026-06-18 JST

Scope: freeze the current release-candidate localization baseline before M10
release candidate builds.

## Result

PASS for the current registry state.

- Frozen locales: `ar`, `en`, `ja`, `zh`, `zhtw`
- Store metadata source locales: `en`, `ja`, `zh`, `zhtw`
- Release-enabled locales: none
- Frozen files: 151
- No registry flags, translation values, store credentials, signing files, or
  upload behavior were changed by this task.

## Evidence Source

The machine-readable freeze manifest is `translation-freeze.json`. It records:

- the enabled app/web language set
- the store metadata source language set
- the M0-T05 preservation policy values
- required validation gates
- SHA-256 and byte-size baselines for frozen translation and metadata files

Validate it with:

```sh
make i18n-freeze-check
```

## Review Policy

After this freeze, any change to a frozen translation or metadata file must be
explicitly reviewed before the freeze manifest is updated.

The update flow is:

```sh
make i18n-check
make i18n-holdouts-check
make i18n-layout-qa-check
make i18n-flag-stages-check
make store-metadata-check
make google-play-metadata-check
make app-store-metadata-check
make screenshot-metadata-check
make android-fastlane-check
make ios-fastlane-check
make release-secret-guardrails-check
make i18n-freeze-manifest
make i18n-freeze-check
```

`make i18n-freeze-manifest` rewrites the manifest from the current workspace.
Use it only after the translation or metadata change has been reviewed and the
required gates pass.

## Guardrails

- `config/languages.json` is frozen as part of the baseline.
- App ARB/settings files are frozen for app-enabled locales.
- Web and API checkout files are frozen for web-enabled locales.
- API catalog files are frozen because they contain language maps.
- Store metadata source and generated metadata files are frozen for the
  current metadata source locales.
- M0-T05 preservation gates are recorded in the manifest review policy.
