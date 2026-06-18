# Stone Signature 68-Language Localization and Release Specification

## 1. Purpose

Stone Signature currently has English, Japanese, and Chinese localization scope
at the product level. The implementation is not yet route-wide: the Flutter app
and Rust web frontend still contain important English/Japanese hard-coded paths,
and any existing Chinese copy or data must be migrated into the new `zh` /
`zhtw` structure instead of being lost. The next target is to support the same
route-wide language set used by finitefield.org: 68 language codes.

This document is an implementation contract for that work. It defines the
language registry, localization file layout, migration plan, validation tools,
release metadata workflow, fastlane introduction, QA gates, rollout strategy,
and rollback expectations.

The goal is not to copy finitefield.org's generator exactly. The goal is a
smaller Stone Signature workflow that makes translation work simple,
repeatable, reviewable, and compatible with future app store releases.

## 2. Goals and Non-Goals

### Goals

- Define one canonical language registry for app, web, API, release metadata,
  and validation tooling.
- Support the finitefield.org route-wide 68 language codes.
- Preserve existing English, Japanese, and Chinese assets during migration.
- Keep English URLs unprefixed and all non-English web URLs under
  `/{route_code}/...`.
- Make Flutter UI strings translatable without editing Dart source for each
  language.
- Make web page copy translatable without adding Askama `if ja else en`
  branches.
- Keep API catalog localization compatible with existing Firestore `*_i18n`
  maps.
- Provide commands that show translation status, generate missing-file reports,
  validate placeholders, and detect unintended English leftovers.
- Prepare app store metadata and screenshots for fastlane `supply` and
  `deliver`.
- Keep release automation safe by separating public metadata from private
  credentials and signing material.
- Preserve existing app navigation architecture: `declarative_nav` and
  `miniriverpod`.
- Preserve repository policy that web and admin screens must not add polling,
  SSE, or WebSocket behavior.

### Non-Goals

- Do not implement translations in this document task.
- Do not introduce a new navigation package such as `go_router`.
- Do not replace Firebase, Firestore, Stripe, Askama, htmx, or ironframe.
- Do not make the admin UI fully localized for external users. Admin can remain
  Japanese/internal unless a later task explicitly changes that.
- Do not upload to Google Play or App Store Connect until metadata-only lanes
  and internal/TestFlight lanes have been tested.
- Do not commit Google Play service account JSON, Apple API keys, keystore
  passwords, or private signing certificates.

## 3. Current Behavior and Files

The repo inspection below calls out the main hard-coded English/Japanese paths.
During implementation, any existing Chinese source strings, catalog values,
store metadata, or manually prepared copy must be imported into `zh` or `zhtw`
before replacing the old structure.

### Flutter App

Relevant files:

- `app/lib/app/localization/hanko_localizations.dart`
- `app/lib/app/app.dart`
- `app/lib/features/settings/presentation/settings_home_screen.dart`
- `app/lib/features/settings/presentation/settings_content.dart`
- `app/lib/features/design/presentation/design_home_screen.dart`
- `app/lib/features/order/presentation/order_flow_entry_screen.dart`
- `app/lib/features/common/data/app_launch_store.dart`
- `app/pubspec.yaml`

Current behavior:

- `HankoLocalizations.supportedLocales` contains only `Locale('en')` and
  `Locale('ja')`.
- All app strings live in one hand-written Dart file.
- Settings language selection is hard-coded to English and Japanese rows.
- Long settings/help/legal content is stored as Dart constants and switches on
  `languageCode == 'ja'`.
- Kanji candidate `reasonLanguage` falls back to `ja` or `en`.
- `MaterialApp` already uses `onGenerateTitle`, `supportedLocales`, and
  `localizationsDelegates`, so the entry point is ready for generated
  localization delegates.
- `app/pubspec.yaml` is the version source and currently has `version:
  1.1.0+11`.

### Web

Relevant files:

- `web/src/main.rs`
- `web/templates/top.html`
- `web/templates/index.html`
- `web/templates/about.html`
- `web/templates/blog_index.html`
- `web/templates/blog_article.html`
- `web/templates/payment_success.html`
- `web/templates/payment_failure.html`
- `web/templates/terms.html`
- `web/templates/commercial_transactions.html`
- `web/blog/articles/*.html`
- `web/static/input.css`
- `web/Makefile`

Current behavior:

- `SUPPORTED_LOCALES` is fixed to `["en", "ja"]`.
- `parse_supported_locale`, `parse_path_locale`, `localized_page_path`, and
  sitemap generation all depend on that fixed list.
- Templates expose `lang_ja_url` and `lang_en_url`, and render only two language
  choices.
- Most page copy is either inline in Askama templates or inline in Rust via
  `localized_text(locale, ja, en)`.
- Blog content is stored as English HTML and Japanese `.ja.html` HTML files.
- Sitemap and `hreflang` output are hard-coded to English and Japanese.

### API and Firestore

Relevant files:

- `api/src/main.rs`
- `api/src/bin/seed_catalog.rs`
- `api/assets/fonts/README.md`
- `api/assets/fonts/profiles.json`
- `api/src/seal_fonts.rs`
- `doc/firebase-firestore-design.md`

Current behavior:

- `/v1/config/public` returns `supported_locales`, `default_locale`,
  `default_currency`, and `currency_by_locale`.
- Default public config supports only `ja` and `en`.
- `api/src/bin/seed_catalog.rs` seeds `app_config/public.supported_locales` as
  `["ja", "en"]`.
- Catalog records already use maps such as `label_i18n`, `description_i18n`,
  `title_i18n`, `story_i18n`, and `photo_alt_i18n`.
- API catalog response code resolves those maps by requested locale with
  fallback.
- Checkout product name formatting is Japanese-vs-English.
- Seal rendering font coverage is separate from UI localization. Seal engraving
  remains limited to approved CJK Han glyph rules.

### Admin

Relevant files:

- `admin/src/main.rs`
- `admin/templates/*.html`
- `admin/static/admin.js`

Current behavior:

- Admin is internal and Japanese-oriented.
- Admin reads and writes localized Firestore maps for some catalog fields.
- Admin should not become the primary 68-language translation UI in this plan.
- Admin must not add polling or streaming.

### Release and Deep Links

Relevant files:

- `app/android/app/build.gradle.kts`
- `app/android/app/src/main/AndroidManifest.xml`
- `app/android/key.properties` if present locally
- `app/ios/Runner/Info.plist`
- `app/ios/Runner.xcodeproj/project.pbxproj`
- `doc/app-release-deep-link-config.md`

Current behavior:

- Android application ID is `org.finitefield.hankofield`.
- iOS bundle identifier is `org.finitefield.hankofield`.
- Android release signing is wired in Gradle and fails clearly when required
  local signing files are missing.
- Deep link and Universal Link paths currently list `/payment/*`,
  `/en/payment/*`, and `/ja/payment/*`.
- App-originated Stripe Checkout returns include `lang`.
- No fastlane files are present.

## 4. Target Language Set

Use these 68 route codes:

```text
ar, az, be, bg, bn, cs, da, de, el, en, es, et, fa, fi, fr, gu, he, hi, hr,
hu, hy, id, is, it, ja, ka, kk, km, kn, ko, ky, lo, lt, lv, mk, ml, mn, mr,
ms, my, ne, nl, no, pa, pl, ps, pt, ro, ru, si, sk, sl, sq, sr, sv, sw, ta,
te, tg, th, tl, tr, uk, ur, uz, vi, zh, zhtw
```

### Special Mapping Rules

| Route code | Meaning | BCP-47 target | Notes |
| --- | --- | --- | --- |
| `en` | English | `en` or `en-US` for stores | Default unprefixed web URL. |
| `ja` | Japanese | `ja` or `ja-JP` for stores | Use `JPY` by default. |
| `zh` | Simplified Chinese | `zh-Hans` or `zh-CN` | Keep route code `zh`. |
| `zhtw` | Traditional Chinese | `zh-Hant` or `zh-TW` | Keep route code `zhtw`. |
| `no` | Norwegian | `no` | Must be quoted if YAML is ever used. Prefer JSON. |
| `ar`, `fa`, `he`, `ps`, `ur` | RTL languages | language-specific | Must render with RTL text direction. |

## 5. Canonical Language Registry

### File

Add:

```text
config/languages.json
```

Use JSON to avoid YAML boolean pitfalls such as `no`.

### Schema

The registry is an array of language entries:

```json
[
  {
    "route_code": "zhtw",
    "bcp47": "zh-Hant",
    "flutter": {
      "languageCode": "zh",
      "scriptCode": "Hant",
      "countryCode": null
    },
    "native_name": "繁體中文",
    "english_name": "Traditional Chinese",
    "text_direction": "ltr",
    "fallback": "en",
    "currency": "USD",
    "web": {
      "enabled": true,
      "indexed": false,
      "url_prefix": "zhtw"
    },
    "app": {
      "enabled": true,
      "selectable": false
    },
    "release": {
      "enabled": false,
      "android_store_locale": "zh-TW",
      "ios_store_locale": "zh-Hant"
    }
  }
]
```

Field rules:

- `route_code` is the stable repo code used in file names and web URLs.
- `bcp47` is the normalized language tag used in HTML `lang`, `hreflang`, API
  metadata, and diagnostics.
- `flutter.languageCode` is required.
- `flutter.scriptCode` is required for script-sensitive locales such as
  `zh-Hans` and `zh-Hant`; otherwise it can be null.
- `flutter.countryCode` should be null unless a region is necessary.
- `text_direction` must be `ltr` or `rtl`.
- `fallback` must point to another `route_code`, usually `en`.
- `currency` is the default pricing currency for this locale. Use `JPY` for
  `ja`; use `USD` for all others unless a business rule says otherwise.
- `web.enabled` controls URL parsing and page rendering.
- `web.indexed` controls sitemap and indexable `hreflang`. Initial rollout can
  render more languages than it indexes.
- `app.enabled` controls whether ARB/assets must exist and whether generated
  localization includes the language.
- `app.selectable` controls whether users can manually choose it in settings.
- `release.enabled` controls whether store metadata must be generated.
- Store locale fields can be null when a platform does not support that locale.

### Registry Consumers

The registry must be consumed by:

- Flutter localization validation.
- Flutter language settings UI.
- Web locale parsing.
- Web language switcher generation.
- Web `hreflang` generation.
- Web sitemap generation.
- API public config seed.
- API locale validation and fallback.
- Translation status and todo commands.
- Store metadata generation.
- fastlane metadata upload lanes.

Do not create separate hard-coded locale lists in app, web, or API after this
registry exists.

## 6. Translation File Layout

### Flutter UI Strings

Add:

```text
app/l10n.yaml
app/lib/l10n/app_en.arb
app/lib/l10n/app_ja.arb
app/lib/l10n/app_zh.arb
app/lib/l10n/app_zh_Hant.arb
```

Final state includes one ARB per `app.enabled` language. ARB file names must use
Flutter locale suffixes, not route codes. For example, route code `zhtw` maps to
Flutter locale `zh_Hant`, so the ARB file is `app_zh_Hant.arb`.

ARB rules:

- `app_en.arb` is the base file.
- Every key in `app_en.arb` must exist in every enabled locale file.
- Every ARB file must map back to exactly one `route_code` in
  `config/languages.json`.
- Placeholders must match exactly across languages.
- ICU plural/select syntax must parse.
- Generated output should be imported by `app/lib/app/app.dart`.
- `MaterialApp` should use generated `localizationsDelegates` and
  `supportedLocales`.
- `onGenerateTitle` must remain localized.

### Flutter Long-Form Content

Move long settings/help/legal content out of Dart constants:

```text
app/assets/i18n/settings/en.json
app/assets/i18n/settings/ja.json
app/assets/i18n/settings/zh.json
app/assets/i18n/settings/zhtw.json
```

This JSON should cover:

- About
- How it works
- FAQ
- Privacy summary
- Terms summary
- Contact/support copy

Rules:

- The JSON shape must be identical for every enabled language.
- The loader must fallback using the registry fallback chain.
- Missing content must show a recoverable localized error, not crash settings.
- Add the assets path to `app/pubspec.yaml`.

### Web Copy

Add:

```text
web/content/i18n/common/en.json
web/content/i18n/top/en.json
web/content/i18n/design/en.json
web/content/i18n/about/en.json
web/content/i18n/blog_index/en.json
web/content/i18n/payment_success/en.json
web/content/i18n/payment_failure/en.json
web/content/i18n/terms/en.json
web/content/i18n/commercial_transactions/en.json
```

Final state includes matching locale files for every `web.enabled` language.

Rules:

- Keep all user-visible strings out of Askama conditionals.
- Rust should deserialize page copy into typed structs.
- Templates should render fields, not branch on locale.
- Shared header/footer/language-switcher copy belongs in `common`.
- Page-specific SEO title, meta description, OG title, and body copy belong in
  page-specific files.

### Web Blog Content

Current files are `.html` and `.ja.html`. Migrate to:

```text
web/content/blog/<slug>/en.html
web/content/blog/<slug>/ja.html
web/content/blog/<slug>/zh.html
web/content/blog/<slug>/metadata.json
```

`metadata.json` should contain language-keyed metadata:

```json
{
  "slug": "hanko-vs-inkan",
  "published_date": "2026-01-01",
  "last_modified_date": "2026-01-01",
  "locales": {
    "en": {
      "title": "...",
      "excerpt": "...",
      "meta_description": "...",
      "image_alt": "..."
    }
  }
}
```

Rules:

- If a blog locale is missing, non-indexed fallback rendering is allowed only
  during migration.
- Indexed blog locale pages must have translated body and metadata.
- Canonical URL should be the localized page URL for localized articles.
- `x-default` should point to English.

### API and Seed Content

Use registry-driven content sources for seed generation:

```text
api/content/i18n/catalog/materials.json
api/content/i18n/catalog/stone_listings.json
api/content/i18n/catalog/facet_tags.json
api/content/i18n/catalog/countries.json
api/content/i18n/checkout/en.json
api/content/i18n/checkout/ja.json
```

The exact file split can change during implementation, but seed data must no
longer require adding new Rust struct fields such as `label_fr` or
`description_fr` for every new language.

## 7. Runtime Behavior

### Locale Selection

App locale priority:

1. Explicit test override passed to `HankoApp`.
2. User-selected preferred language from `AppLaunchStore`.
3. Platform locale if it maps to an `app.enabled` registry entry.
4. Registry default, initially `en` for UI fallback unless product policy
   changes.

Web locale priority:

1. Path prefix, for example `/fr/about`.
2. `lang` query parameter where currently supported for checkout/payment
   compatibility.
3. Configured `HANKO_WEB_LOCALE`.
4. Registry default.

API locale priority:

1. Request locale parameter or request body locale.
2. Contact preferred locale when order context requires it.
3. Public config default locale.
4. Registry fallback chain.

### Fallback Order

For any localized map or content file:

1. Requested route code.
2. `fallback` from `config/languages.json`.
3. Default route code.
4. `en`.
5. First non-empty value.

Fallback use must be observable in diagnostics. A release-enabled locale should
not silently fallback for user-visible copy unless the key is registered in an
intention sidecar.

### URL Rules

- English stays unprefixed: `/`, `/about`, `/blog/<slug>`.
- Every non-English locale is prefixed:
  `/ja/`, `/fr/about`, `/zhtw/blog/<slug>`.
- `/en/...` may remain accepted for compatibility and redirects, but canonical
  English URLs must be unprefixed.
- Unknown locale prefixes return 404, not English content.
- `zhtw` remains the path prefix for Traditional Chinese.

### Deep Link Rules

Update `doc/app-release-deep-link-config.md` and platform association files
when localized payment paths expand.

Required behavior:

- Custom scheme remains the primary app-originated return:
  `hankofield://checkout/*`.
- Universal Links/App Links must allow all release-enabled web payment prefixes
  when app links are used.
- Existing `/payment/*`, `/en/payment/*`, and `/ja/payment/*` routes must remain
  compatible.
- Checkout return parsing must continue accepting both `lang` and `locale`.

## 8. UI and Accessibility Requirements

### App Language Settings

The language settings screen must:

- Render from `config/languages.json`.
- Show `native_name` and optionally `english_name`.
- Mark the selected language.
- Exclude languages where `app.selectable` is false.
- Preserve the existing selected locale if a newly added language is not yet
  selectable.
- Save `route_code`, not platform store locale.
- Recover gracefully if an old saved locale is no longer enabled.

### Web Language Switcher

The language switcher must:

- Render from `web.enabled` language entries.
- Use `native_name`.
- Set `hreflang` to `bcp47`.
- Mark the current locale.
- Link to the same page in the target locale when translated.
- Link to the locale home page or hide the link if a page is not enabled.
- Remain keyboard accessible.

### RTL

For RTL languages:

- Flutter must use `TextDirection.rtl`.
- HTML must set `dir="rtl"` on `<html>` or a top-level container.
- Icons with directional meaning must be reviewed.
- Numeric values, currency, order IDs, email addresses, and URLs must remain
  readable.

### Text Overflow

All screens must tolerate longer translations:

- Avoid fixed one-line button labels where a translated phrase can wrap.
- Avoid fixed-width label columns for localized labels.
- Check compact mobile widths.
- Check checkout, settings, catalog filters, and payment result pages first.

## 9. Admin and Data Entry Requirements

Admin remains an internal Japanese-first tool in this plan, but it must not
damage multilingual data.

Requirements:

- Existing admin save flows must preserve unknown keys in `*_i18n` maps.
- Editing Japanese or English fields must not drop `fr`, `zh`, `zhtw`, or other
  translated values.
- If admin adds UI for multi-language fields, it should start with a compact
  "localized values" editor driven by the registry, not 68 always-visible form
  fields.
- Admin list/detail pages may continue showing Japanese labels by default.
- Admin must not add polling, SSE, or WebSocket behavior.

## 10. Tooling Commands

Add these Make targets at the repository root:

```text
make i18n-generate
make i18n-status
make i18n-todo
make i18n-check
make i18n-check LANGS=zh,zhtw
make i18n-check FILE=app/lib/l10n/app_fr.arb
make i18n-export
make i18n-import
make release-metadata-generate
make release-metadata-check
```

Implementation can live under:

```text
scripts/i18n/
  languages_check.rs or languages_check.py
  i18n_status.rs or i18n_status.py
  i18n_todo.rs or i18n_todo.py
  i18n_check.rs or i18n_check.py
  release_metadata_generate.rs or release_metadata_generate.py
```

Use Rust if the checks need to share parsing code with `api` or `web`; use
Python if it keeps the scripts materially smaller. The first implementation can
be script-based and later promoted into a crate if needed.

### `make i18n-status`

Output:

- total registry languages
- enabled app languages
- enabled web languages
- release-enabled store languages
- missing app ARB files
- missing app settings JSON files
- missing web page copy files
- missing store metadata files
- fallback usage summary

This command must be read-only.

### `make i18n-todo`

Output:

- file path
- locale
- missing key
- base English value
- current fallback value if any
- suggested sidecar path

Options:

- `LANGS=fr,de`
- `FILE=<path>`
- `OUT=<path>`

### `make i18n-check`

Checks:

- `config/languages.json` parses and contains unique route codes.
- Every fallback points to an existing route code.
- Every route code has valid BCP-47 output metadata.
- RTL languages are marked correctly.
- App ARB files have identical keys.
- ARB placeholders and metadata match base.
- App settings JSON shape matches base.
- Web copy JSON shape matches base.
- Blog metadata and body exist for indexed languages.
- API seed content contains required locale maps or intentional fallback
  declarations.
- Non-English values do not equal English unless declared.
- Intention sidecars use allowed reason codes.
- Store metadata source exists for release-enabled languages.

Exit code:

- `0` means clean.
- non-zero means missing, malformed, or unapproved fallback.

### Intention Sidecars

Use sidecars for intentional English or shared values:

```text
app/lib/l10n/app_fr_intentions.json
app/assets/i18n/settings/fr_intentions.json
web/content/i18n/top/fr_intentions.json
api/content/i18n/catalog/fr_intentions.json
release/store_metadata/source/fr_intentions.json
```

Allowed reason codes:

- `brand_name`
- `legal_entity_name`
- `product_name`
- `technical_identifier`
- `url`
- `email`
- `currency_code`
- `country_code`
- `kanji_character`
- `code_literal`
- `font_name`
- `law_name`
- `intentionally_english`

Sidecar shape:

```json
{
  "appTitle": "brand_name",
  "support.emailLabel": "email"
}
```

## 11. Subsystem Scope Reference

This section summarizes the work by subsystem. It is not the execution
checklist. Use Section 12 for the strict task order, starting from `M0-T01`.

### Foundation Registry

Files likely touched:

- `config/languages.json`
- `Makefile`
- `scripts/i18n/*`
- `doc/multilingual-release-plan.md`

Scope items:

- Add `config/languages.json` with all 68 route codes.
- Mark `web.enabled`, `app.enabled`, and `release.enabled` separately.
- Set `release.enabled` false for all newly added languages until store
  metadata and screenshots are ready.
- Implement registry validation.
- Add `make i18n-status`.
- Add tests or snapshot fixtures for `zh`, `zhtw`, `no`, and RTL entries.

Acceptance criteria:

- `make i18n-status` lists all 68 route codes.
- Duplicate route codes fail validation.
- Invalid fallback codes fail validation.
- `no` remains a string route code.
- `zhtw` resolves to Traditional Chinese platform metadata.

### Flutter Generated Localization

Files likely touched:

- `app/l10n.yaml`
- `app/lib/l10n/*.arb`
- `app/lib/app/app.dart`
- `app/lib/app/localization/*`
- `app/lib/features/settings/presentation/settings_home_screen.dart`
- `app/lib/features/settings/presentation/settings_content.dart`
- `app/lib/features/common/data/app_launch_store.dart`
- `app/pubspec.yaml`
- `app/test/widget_test.dart`

Scope items:

- Create base `app_en.arb` from existing English strings.
- Create `app_ja.arb` from existing Japanese strings.
- Configure Flutter `gen-l10n`.
- Replace hand-written localization accessors with generated accessors.
- Keep a temporary compatibility wrapper only if it materially reduces the
  migration risk.
- Move long settings content to JSON assets.
- Load language settings rows from the registry.
- Store preferred `route_code`.
- Add fallback for old saved `en` and `ja` values.
- Add RTL handling.
- Add UI tests for `en`, `ja`, `zh`, `zhtw`, and one RTL locale once those
  files exist.

Acceptance criteria:

- `flutter gen-l10n` or `flutter pub get` generates localization output.
- `flutter test` passes.
- English and Japanese app text remains equivalent to current behavior.
- Language settings no longer contains hard-coded English/Japanese rows.
- Missing settings JSON shows a recoverable error state or falls back without
  crashing.

### Web Copy Extraction

Files likely touched:

- `web/src/main.rs`
- `web/templates/*.html`
- `web/content/i18n/**/*.json`
- `web/blog/articles/*`
- `web/content/blog/**/*`
- `web/Makefile`

Scope items:

- Add language registry loader for `web`.
- Replace `SUPPORTED_LOCALES` with registry-backed validation.
- Introduce `LanguageLink` and page copy structs.
- Extract `top`, `index/design`, `about`, `blog_index`, `payment_success`,
  `payment_failure`, `terms`, and `commercial_transactions` copy to JSON.
- Remove `lang_ja_url` and `lang_en_url` fields from templates.
- Replace hard-coded `hreflang` tags with a loop over language links.
- Generate sitemap entries from indexed registry languages.
- Migrate blog article metadata and bodies to language-keyed content.
- Preserve current English and Japanese URLs.
- Keep `/en/...` compatibility behavior if existing inbound links require
  it, but do not make `/en/...` canonical.

Acceptance criteria:

- `cargo test` for `web` passes.
- Unknown locale path returns 404.
- `/about` renders English canonical.
- `/ja/about` renders Japanese canonical.
- A non-indexed enabled locale can render for QA without appearing in sitemap.
- `hreflang` includes all indexed locales and `x-default`.
- Templates no longer contain `if selected_locale == "ja"` for user-visible
  copy.

### API, Firestore Seed, and Checkout

Files likely touched:

- `api/src/main.rs`
- `api/src/bin/seed_catalog.rs`
- `api/content/i18n/**/*`
- `doc/firebase-firestore-design.md`
- `doc/app-release-deep-link-config.md`

Scope items:

- Load or generate public config supported locales from the registry.
- Replace hard-coded seed locale arrays with registry data.
- Move catalog seed text from Rust structs to data files or map-based seed
  structures.
- Preserve unknown locale keys when reading/writing Firestore maps.
- Extend checkout product labels to data-driven localized templates.
- Define `reason_language` mapping for Gemini prompts:
  - supported languages use their BCP-47 code if prompt quality is acceptable
  - unsupported prompt languages fallback to English
  - fallback must be visible in response diagnostics
- Add tests for `zh`, `zhtw`, unsupported locale rejection, and fallback.

Acceptance criteria:

- `/v1/config/public` returns registry-driven supported locales.
- Catalog requests for a supported locale do not fail.
- Missing catalog values fallback predictably.
- Checkout URLs preserve selected `lang`.
- Checkout product names are data-driven for at least `en`, `ja`, `zh`, and
  `zhtw` before wider rollout.

### Admin Data Preservation

Files likely touched:

- `admin/src/main.rs`
- `admin/templates/material_*`
- `admin/templates/stone_listing_*`
- `admin/templates/country_*`
- `admin/templates/facet_tag_*`

Scope items:

- Audit every admin form that writes a Firestore `*_i18n` map.
- Ensure saves merge edited fields into existing maps instead of replacing
  maps with only `ja` / `en`.
- Add tests for preserving unknown locale keys.
- Optionally add a collapsed localized-values editor for catalog records.

Acceptance criteria:

- Editing a Japanese material label preserves existing `fr`, `zh`, and `zhtw`
  values.
- Admin remains usable without rendering 68 visible inputs by default.
- No admin polling, SSE, or WebSocket behavior is added.

### Translation Tooling

Files likely touched:

- `Makefile`
- `scripts/i18n/*`
- `app/lib/l10n/*`
- `app/assets/i18n/**/*`
- `web/content/i18n/**/*`
- `api/content/i18n/**/*`
- `release/store_metadata/source/*`

Scope items:

- Implement `make i18n-todo`.
- Implement `make i18n-check`.
- Implement sidecar validation.
- Implement placeholder/ICU validation for ARB.
- Implement JSON shape validation for long-form content.
- Implement English-leftover checks.
- Add CI target once checks are stable.

Acceptance criteria:

- Missing locale files are reported with actionable paths.
- Placeholder mismatch fails the check.
- Unapproved English leftovers fail the check for non-English locales.
- Approved sidecar entries suppress only the intended key.
- `LANGS=` and `FILE=` filters work.

### 68-Language Content Rollout

Files likely touched:

- all localization content paths
- release metadata source paths

Scope items:

- Translate base app ARB files.
- Translate app long-form settings files.
- Translate web page copy.
- Translate blog metadata and bodies for indexed languages.
- Translate catalog and checkout seed content.
- Translate store metadata.
- Run tiered QA.
- Enable languages in stages:
  - render-only
  - app-selectable
  - web-indexed
  - store-release-enabled

Acceptance criteria:

- Each stage has a registry flag change and validation evidence.
- No language becomes store-release-enabled without metadata and screenshot
  readiness.
- No indexed web language has untranslated page title, meta description, or
  primary body copy.

### Store Metadata and fastlane

Files likely touched:

- `release/store_metadata/source/*.json`
- `release/store_metadata/google_play/**`
- `release/store_metadata/app_store/**`
- `app/android/Gemfile`
- `app/android/Gemfile.lock`
- `app/android/fastlane/Appfile`
- `app/android/fastlane/Fastfile`
- `app/ios/Gemfile`
- `app/ios/Gemfile.lock`
- `app/ios/fastlane/Appfile`
- `app/ios/fastlane/Fastfile`
- `.gitignore`
- `doc/app-release-deep-link-config.md`

Scope items:

- Add source store metadata JSON per release-enabled language.
- Generate Google Play metadata folders.
- Generate App Store metadata folders.
- Add fastlane with Bundler.
- Add metadata-only Android lane.
- Add metadata-only iOS lane.
- Add Google Play internal lane.
- Add TestFlight lane.
- Add production lanes only after internal lanes are proven.
- Add `.gitignore` entries for private service account JSON, Apple API key
  files, exported `.ipa`, generated keystores, and local fastlane reports if
  needed.

Acceptance criteria:

- `bundle exec fastlane metadata` can run from `app/android` and `app/ios`
  without uploading binaries.
- Internal Google Play lane uploads an AAB to internal testing.
- TestFlight lane uploads a signed IPA.
- Production lanes require explicit manual confirmation.
- Store metadata generation validates unsupported store locales before upload.

## 12. Detailed Delivery Milestones and Tasks

This section is the canonical execution checklist. Start at `M0-T01` and move
down the page in order. Do not start the next milestone until the current
milestone's exit gate is satisfied, unless a later task is only a read-only
investigation that does not change code, content, release metadata, or
production configuration.

Keep task IDs stable when creating GitHub issues or PR checklists.

A task is complete only when:

- the implementation or document change is merged
- relevant tests or validation commands have passing evidence
- unrelated dirty worktree changes are not included
- rollout or rollback notes are updated when user-visible behavior changes

### Milestone Dependency Order

| Milestone | Purpose | Depends on | Exit gate |
| --- | --- | --- | --- |
| M0 | Baseline inventory and migration safety | none | Existing English, Japanese, and Chinese assets are inventoried. |
| M1 | Registry and read-only tooling | M0 | `make i18n-status` reads the 68-language registry. |
| M2 | Flutter app migration | M1 | App uses generated localization for existing languages. |
| M3 | Web copy and route migration | M2 | Web renders registry-backed localized pages. |
| M4 | API, catalog, and checkout localization | M3 | Public config, catalog, and checkout are registry-backed. |
| M5 | Admin data preservation | M4 | Admin saves do not drop unknown locale keys. |
| M6 | Translation workflow tooling | M5 | `make i18n-check` blocks missing or unsafe translations. |
| M7 | Pilot language rollout | M6 | `zh`, `zhtw`, and one RTL locale pass render QA. |
| M8 | Store metadata and fastlane | M7 | Metadata-only fastlane lanes pass. |
| M9 | 68-language content production | M8 | Target language batches pass translation checks. |
| M10 | Release QA and staged launch | M9 | Internal/TestFlight release evidence is recorded. |
| M11 | Post-release monitoring and cleanup | M10 | Locale diagnostics and support runbook are ready. |

### M0: Baseline Inventory and Migration Safety

- [x] `M0-T01` Inventory existing app strings.
  Output: a short inventory of Dart localization keys, long settings content,
  and any existing Chinese app copy.
  Done when: every source file that will be migrated to ARB or JSON is listed.
- [x] `M0-T02` Inventory existing web copy and blog content.
  Output: page-by-page list of template strings, Rust inline copy, blog files,
  SEO metadata, and current `hreflang` behavior.
  Done when: every web route in the current sitemap has a migration target.
- [x] `M0-T03` Inventory API, catalog, checkout, and Firestore locale data.
  Output: list of `*_i18n` maps, seed constants, checkout labels, and fallback
  rules.
  Done when: no catalog or checkout copy is left without an owner path.
- [x] `M0-T04` Inventory release metadata and deep-link state.
  Output: current app identifiers, version source, signing assumptions,
  checkout return paths, and store copy locations.
  Done when: fastlane setup can proceed without rediscovering release basics.
- [ ] `M0-T05` Record migration safety rules.
  Output: checklist for preserving existing English, Japanese, and Chinese
  content during each migration PR.
  Done when: the checklist is referenced from later migration issues.

#### M0-T01 App String Inventory

Completed on 2026-06-18. This inventory covers Flutter app strings only. It
identifies every current source file whose user-visible app copy must move to
ARB or JSON during `M2`.

ARB migration sources:

| Source file | Current content | Migration target | Notes |
| --- | --- | --- | --- |
| `app/lib/app/localization/hanko_localizations.dart` | Hand-written `HankoLocalizations`, fixed `supportedLocales`, 387 public string getters, `_HankoStrings`, `_localizedValues`, English values, Japanese values, and `settingsVersionMessage(String version)` placeholder replacement. | `app/lib/l10n/app_en.arb`, `app/lib/l10n/app_ja.arb`, generated localizations, and temporary compatibility wrapper if needed. | This is the primary short UI string source. Preserve key names where practical so existing widget usage can be migrated mechanically. |
| `app/lib/features/design/presentation/design_home_screen.dart` | Inline localized tip prefix: `Tip: ` / `ヒント: `. Also contains `reason_language` mapping from UI locale to `ja` or `en`. | ARB key for the tip prefix. Registry-backed reason-language adapter, not an ARB text key. | The tip prefix is user-visible copy outside `HankoLocalizations` and must not be missed. |
| `app/lib/features/order_lookup/presentation/order_lookup_entry_screen.dart` | Inline localized status labels in `_statusLabel` for payment, production, shipping, and fulfillment values such as `paid`, `pending_payment`, `in_production`, `shipped`, and `fulfilled`. | ARB keys or ICU/select-backed helper for status labels. | These labels are currently English/Japanese ternaries and must become generated localization keys before more locales are enabled. |

Long-form JSON migration sources:

| Source file | Current content | Migration target | Notes |
| --- | --- | --- | --- |
| `app/lib/features/settings/presentation/settings_content.dart` | `_enContent` and `_jaContent` for About, How it works, FAQ, Privacy, Terms, and Contact. Includes headings, paragraph bodies, FAQ items, legal sections, official URLs, contact URL, and support email. | `app/assets/i18n/settings/en.json`, `app/assets/i18n/settings/ja.json`, and matching files for later enabled languages. | Keep URLs, email address, brand names, legal entity names, and governing-law text eligible for intention sidecars rather than forcing translation. |

Integration files that contain locale wiring but are not text sources:

- `app/lib/app/localization/app_localization.dart` only re-exports the current
  localization file. It should be updated or removed after generated
  localizations are introduced, but it has no strings to migrate.
- `app/lib/app/app.dart` wires `HankoLocalizations.supportedLocales`,
  `localizationsDelegates`, `onGenerateTitle`, saved locale selection, and
  `_reasonLanguageForCurrentLocale`. It needs registry/generated-localization
  integration in `M2`, but no ARB/JSON content originates here.
- `app/lib/features/settings/presentation/settings_home_screen.dart` renders
  hard-coded English/Japanese language rows from existing localization keys. It
  should render registry-driven language rows in `M2`, but the label source is
  currently `hanko_localizations.dart`.
- `app/test/widget_test.dart` contains locale-specific expectations and helper
  `MaterialApp` setup using `HankoLocalizations.supportedLocales` and
  `localizationsDelegates`. Update these tests with the `M2` migration, but do
  not treat test literals as source translation copy.

Existing Chinese app copy:

- No standalone `zh` or `zhtw` Flutter locale file, map, or asset exists in the
  current app source.
- Existing Chinese-related UI labels are language-style labels inside
  `hanko_localizations.dart`: `designKanjiStyleChinese` and
  `designKanjiStyleTaiwanese` in English and Japanese.
- Existing Chinese-related long-form references are descriptive Japanese/English
  content in `settings_content.dart`, such as shipping from the partner workshop
  in China. These are not Chinese translations and should migrate as part of
  `en` / `ja` settings JSON.

#### M0-T02 Web Copy and Blog Inventory

Completed on 2026-06-18. This inventory covers the Rust web frontend, Askama
templates, browser-side dynamic copy, blog article files, SEO metadata, and
current sitemap / `hreflang` behavior. Every route that is currently emitted by
the sitemap has a migration target below.

Current web locale behavior:

- `web/src/main.rs` uses `SUPPORTED_LOCALES: &["en", "ja"]`.
- `parse_supported_locale`, `parse_path_locale`, `localized_page_path`,
  `localized_page_url`, language-switcher URL fields, sitemap generation, and
  `hreflang` output are all tied to English and Japanese.
- `localized_text(locale, ja, en)` owns page-level inline Rust copy such as SEO
  title, meta description, purchase notes, and noindex payment-page metadata.
- Askama templates render many user-visible strings through
  `{% if selected_locale == "ja" %}` branches.
- Sitemap entries currently emit English and Japanese alternates plus
  `x-default`. English canonical URLs are unprefixed.

Web route migration targets:

| Current route | Current source files and copy owners | Current SEO / `hreflang` behavior | Migration target |
| --- | --- | --- | --- |
| `/`, `/{locale}`, `/{locale}/` | `render_top_page`, `web/templates/top.html`, blog card metadata from `web/blog/articles/*.html`. | Indexed. English canonical is `/`; Japanese alternate is `/ja/`; `x-default` points to English. Page title and meta description come from `localized_text`. | `web/content/i18n/top/<lang>.json`, shared header/footer/language-switcher copy in `web/content/i18n/common/<lang>.json`, and blog card metadata from migrated blog metadata. |
| `/about`, `/{locale}/about` | `render_about_page`, `web/templates/about.html`. | Indexed. English canonical is `/about`; Japanese alternate is `/ja/about`; `x-default` points to English. Page title and meta description come from `localized_text`. | `web/content/i18n/about/<lang>.json` plus `common/<lang>.json`. |
| `/design`, `/{locale}/design` | `render_design_page`, `web/templates/index.html`, `web/templates/kanji_suggestions.html`, `web/templates/purchase_result.html`, `web/static/app.js`, API catalog labels. | Indexed. English canonical is `/design`; Japanese alternate is `/ja/design`; `x-default` points to English. Page title, meta description, and purchase note come from `localized_text`. | `web/content/i18n/design/<lang>.json` for page and JavaScript dynamic copy, plus fragment copy either in `design/<lang>.json` or dedicated `kanji_suggestions/<lang>.json` and `purchase_result/<lang>.json`. API catalog labels remain catalog-owned. |
| `/blog`, `/{locale}/blog` | `render_blog_index_page`, `web/templates/blog_index.html`, blog front matter card fields. | Indexed. English canonical is `/blog`; Japanese alternate is `/ja/blog`; `x-default` points to English. Page title and meta description come from `localized_text`; card metadata comes from article front matter. | `web/content/i18n/blog_index/<lang>.json`, `common/<lang>.json`, and migrated blog `metadata.json` files. |
| `/blog/{slug}`, `/{locale}/blog/{slug}` | `render_blog_article_page`, `web/templates/blog_article.html`, English article body files, Japanese `.ja.html` body files, and front matter metadata. | Indexed for each current blog slug. Canonical URL is the localized article URL. Alternates are English, Japanese, and `x-default` English. | `web/content/blog/<slug>/<lang>.html` and `web/content/blog/<slug>/metadata.json` with language-keyed title, excerpt, meta description, date display, and image alt text. |
| `/terms`, `/{locale}/terms` | `render_terms_page`, `web/templates/terms.html`. | Indexed. English canonical is `/terms`; Japanese alternate is `/ja/terms`; `x-default` points to English. Page title and meta description come from `localized_text`. | `web/content/i18n/terms/<lang>.json` plus intention sidecars for legal entity names, addresses, governing-law terms, URLs, and other values that should intentionally remain unchanged. |
| `/commercial-transactions`, `/{locale}/commercial-transactions` | `render_commercial_transactions_page`, `web/templates/commercial_transactions.html`. | Indexed. English canonical is `/commercial-transactions`; Japanese alternate is `/ja/commercial-transactions`; `x-default` points to English. Page title and meta description come from `localized_text`. | `web/content/i18n/commercial_transactions/<lang>.json` plus intention sidecars for seller name, representative name, address, phone, email, production origin, payment provider names, and legal labels that should intentionally remain unchanged. |
| `/payment/success`, `/{locale}/payment/success` | `render_payment_success_page`, `web/templates/payment_success.html`. | Not in the sitemap. Rendered with `noindex`; page title and meta description come from `localized_text`. | `web/content/i18n/payment_success/<lang>.json` plus `common/<lang>.json`. Keep localized paths compatible for checkout return URLs. |
| `/payment/failure`, `/{locale}/payment/failure` | `render_payment_failure_page`, `web/templates/payment_failure.html`. | Not in the sitemap. Rendered with `noindex`; page title and meta description come from `localized_text`. | `web/content/i18n/payment_failure/<lang>.json` plus `common/<lang>.json`. Keep localized paths compatible for checkout return URLs. |
| `/sitemap.xml` | `handle_sitemap_xml`, `build_sitemap_xml`, `sitemap_url_entry`, `SITEMAP_STATIC_PAGES`, and loaded blog post metadata. | Emits English and Japanese `<url>` entries for indexed static pages, the blog index, and every blog article. `hreflang` values are hard-coded to `en`, `ja`, and `x-default`. | Registry-driven sitemap builder using `web.indexed`, route-code to BCP-47 mapping, localized canonical URLs, and fallback exclusion for non-indexed QA languages. |
| `/robots.txt` | `handle_robots_txt`, `build_robots_txt`, and sitemap URL generation. | No localized route and no page copy. Points crawlers to `/sitemap.xml`. | No translation file needed. Keep sitemap URL generation aligned with the registry-backed web base URL. |
| `/kanji`, `/mock/kanji` | `handle_kanji_suggestions`, `web/templates/kanji_suggestions.html`. | Fragment endpoints; not in sitemap. Locale comes from request context or mock path. | Fragment copy in `web/content/i18n/design/<lang>.json` or `web/content/i18n/kanji_suggestions/<lang>.json`. |
| `/purchase`, `/mock/purchase` | `handle_purchase_impl`, `web/templates/purchase_result.html`. | Fragment endpoints; not in sitemap. Locale comes from request context or mock path. | Fragment copy in `web/content/i18n/design/<lang>.json` or `web/content/i18n/purchase_result/<lang>.json`. |

Template copy inventory:

| Template | Current copy pattern | Migration target |
| --- | --- | --- |
| `web/templates/top.html` | 11 Japanese-locale conditional branches. | `top/<lang>.json` and `common/<lang>.json`. |
| `web/templates/index.html` | 91 Japanese-locale conditional branches for the design page, filters, seal preview, checkout form, and supporting text. | `design/<lang>.json`, common copy, and fragment JSON where useful. |
| `web/templates/about.html` | 19 Japanese-locale conditional branches. | `about/<lang>.json`. |
| `web/templates/blog_index.html` | 7 Japanese-locale conditional branches. | `blog_index/<lang>.json` and blog metadata. |
| `web/templates/blog_article.html` | 7 Japanese-locale conditional branches around article chrome and navigation. | `common/<lang>.json`, blog chrome fields, and blog metadata. |
| `web/templates/payment_success.html` | 26 Japanese-locale conditional branches. | `payment_success/<lang>.json`. |
| `web/templates/payment_failure.html` | 24 Japanese-locale conditional branches. | `payment_failure/<lang>.json`. |
| `web/templates/terms.html` | 38 Japanese-locale conditional branches. | `terms/<lang>.json` plus legal intention sidecars. |
| `web/templates/commercial_transactions.html` | 46 Japanese-locale conditional branches. | `commercial_transactions/<lang>.json` plus legal intention sidecars. |
| `web/templates/kanji_suggestions.html` | 5 Japanese-locale conditional branches. | `design/<lang>.json` or `kanji_suggestions/<lang>.json`. |
| `web/templates/purchase_result.html` | 7 Japanese-locale conditional branches. | `design/<lang>.json` or `purchase_result/<lang>.json`. |

Rust inline copy and route owner inventory:

- `TopPageTemplate`, `AboutTemplate`, `PageTemplate`, `BlogIndexTemplate`,
  `BlogArticleTemplate`, `PaymentSuccessTemplate`, `PaymentFailureTemplate`,
  `CommercialTransactionsTemplate`, and `TermsTemplate` carry localized SEO,
  language-switcher URLs, and template fields.
- `render_top_page`, `render_about_page`, `render_design_page`,
  `render_blog_index_page`, `render_payment_success_page`,
  `render_payment_failure_page`, `render_commercial_transactions_page`, and
  `render_terms_page` contain page-level `localized_text` calls that should
  move into page JSON files.
- `render_blog_article_page` resolves language-specific article bodies and
  metadata from `BlogPost`.
- `handle_purchase_impl` and checkout validation helpers contain inline
  `localized_text` error messages for form validation and purchase failures.
  Move these to checkout or purchase-result JSON with stable keys.
- `SUPPORTED_LOCALES`, `parse_supported_locale`, `parse_path_locale`,
  `localized_page_path`, `localized_page_url`, `top_url`, `design_url`,
  `blog_index_url`, `blog_article_url`, and `sitemap_url_entry` are the main
  route and SEO functions that must become registry-backed.
- `web/static/app.js` owns browser-side user-visible copy for filter summaries,
  empty states, form validation, purchase submit states, htmx errors, Kanji
  reading labels, selected Kanji messages, preview shape labels, and fallback
  summary text. This copy must move to JSON consumed by the design page so
  adding a language does not require editing JavaScript source.

Blog content inventory:

| Slug | Current files | Current metadata owner | Migration target |
| --- | --- | --- | --- |
| `chinese-chop-seal-vs-japanese-hanko` | `web/blog/articles/chinese-chop-seal-vs-japanese-hanko.html`, `web/blog/articles/chinese-chop-seal-vs-japanese-hanko.ja.html` | English front matter with Japanese fields. | `web/content/blog/chinese-chop-seal-vs-japanese-hanko/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `custom-jade-seal` | `web/blog/articles/custom-jade-seal.html`, `web/blog/articles/custom-jade-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/custom-jade-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `custom-stone-seal-gift` | `web/blog/articles/custom-stone-seal-gift.html`, `web/blog/articles/custom-stone-seal-gift.ja.html` | English front matter with Japanese fields. | `web/content/blog/custom-stone-seal-gift/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `english-name-kanji-seal` | `web/blog/articles/english-name-kanji-seal.html`, `web/blog/articles/english-name-kanji-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/english-name-kanji-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `hanko-vs-inkan` | `web/blog/articles/hanko-vs-inkan.html`, `web/blog/articles/hanko-vs-inkan.ja.html` | English front matter with Japanese fields. | `web/content/blog/hanko-vs-inkan/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `how-to-choose-stone-seal` | `web/blog/articles/how-to-choose-stone-seal.html`, `web/blog/articles/how-to-choose-stone-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/how-to-choose-stone-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `jade-agate-qingtian-stone-seal` | `web/blog/articles/jade-agate-qingtian-stone-seal.html`, `web/blog/articles/jade-agate-qingtian-stone-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/jade-agate-qingtian-stone-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `japanese-hanko-souvenir` | `web/blog/articles/japanese-hanko-souvenir.html`, `web/blog/articles/japanese-hanko-souvenir.ja.html` | English front matter with Japanese fields. | `web/content/blog/japanese-hanko-souvenir/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `luxury-personal-seal` | `web/blog/articles/luxury-personal-seal.html`, `web/blog/articles/luxury-personal-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/luxury-personal-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `one-of-a-kind-stone-seal` | `web/blog/articles/one-of-a-kind-stone-seal.html`, `web/blog/articles/one-of-a-kind-stone-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/one-of-a-kind-stone-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `personal-seal-symbol-of-identity` | `web/blog/articles/personal-seal-symbol-of-identity.html`, `web/blog/articles/personal-seal-symbol-of-identity.ja.html` | English front matter with Japanese fields. | `web/content/blog/personal-seal-symbol-of-identity/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `personal-seals-for-artists` | `web/blog/articles/personal-seals-for-artists.html`, `web/blog/articles/personal-seals-for-artists.ja.html` | English front matter with Japanese fields. | `web/content/blog/personal-seals-for-artists/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `what-is-a-hanko` | `web/blog/articles/what-is-a-hanko.html`, `web/blog/articles/what-is-a-hanko.ja.html` | English front matter with Japanese fields. | `web/content/blog/what-is-a-hanko/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `what-is-a-personal-seal` | `web/blog/articles/what-is-a-personal-seal.html`, `web/blog/articles/what-is-a-personal-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/what-is-a-personal-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |
| `what-to-engrave-on-seal` | `web/blog/articles/what-to-engrave-on-seal.html`, `web/blog/articles/what-to-engrave-on-seal.ja.html` | English front matter with Japanese fields. | `web/content/blog/what-to-engrave-on-seal/en.html`, `ja.html`, future locale HTML files, and `metadata.json`. |

Current blog migration notes:

- Each current slug has an English article file and a Japanese `.ja.html`
  article file.
- Current English front matter owns `title`, `excerpt`, `date`,
  `date_display`, `meta_description`, `image_url`, and `image_alt`.
- Current English front matter also owns Japanese metadata fields such as
  `title_ja`, `excerpt_ja`, `date_display_ja`, `meta_description_ja`, and
  `image_alt_ja`.
- The existing article bodies are parallel English/Japanese files, not a
  directory-based locale structure.
- No standalone `zh` or `zhtw` web article translation exists today. Chinese
  subject matter in slugs such as `chinese-chop-seal-vs-japanese-hanko`,
  `english-name-kanji-seal`, `custom-jade-seal`, and
  `jade-agate-qingtian-stone-seal` is topical content, not Chinese locale copy.

Sitemap coverage and migration targets:

- Current indexed static sitemap routes are `/`, `/about`, `/design`,
  `/terms`, and `/commercial-transactions`, with English and Japanese URL
  entries for each route.
- `/blog` is indexed with English and Japanese URL entries.
- All 15 blog article slugs above are indexed with English and Japanese URL
  entries.
- `/payment/success` and `/payment/failure` are localized user-visible pages
  but are intentionally excluded from the sitemap and rendered as `noindex`.
- `/kanji`, `/purchase`, `/mock/kanji`, `/mock/purchase`, admin proxy routes,
  and static asset routes are not sitemap routes. They still need localized
  fragment copy where they render user-visible UI.
- The migration target for sitemap and `hreflang` behavior is the M3 registry
  loader plus page availability checks. Non-indexed QA languages must be
  renderable without being emitted in `/sitemap.xml`.

#### M0-T03 API, Catalog, Checkout, and Firestore Locale Inventory

Completed on 2026-06-18. This inventory covers Rust API runtime locale
resolution, Firestore `*_i18n` maps, seed constants, checkout labels, Stripe
Checkout locale propagation, and admin-managed catalog writes. Every current
catalog or checkout copy source has an owner path below.

Current API locale behavior:

- `api/src/main.rs` owns `PublicConfig`, catalog response shaping, order
  creation, Stripe Checkout session creation, Gemini Kanji candidate prompt
  language handling, and fallback resolution.
- `app_config/public` currently resolves to `supported_locales=["ja","en"]`,
  `default_locale="ja"`, `default_currency="USD"`, and
  `currency_by_locale={"ja":"JPY","en":"USD"}`.
- `default_public_config()` is the runtime fallback when Firestore config is
  missing or incomplete. `api/src/bin/seed_catalog.rs` writes the same locale
  and currency policy into Firestore.
- API catalog endpoints reject unsupported `locale` values after loading public
  config. Order creation rejects unsupported `orders.locale` and
  `contact.preferred_locale`.
- `resolve_localized(values, requested_locale, default_locale)` falls back in
  this order: requested locale, default locale, `ja`, then the first non-empty
  value by sorted key.
- `lookup_locale` accepts exact locale keys and base-language lookup for
  hyphenated BCP-47 values.

Firestore locale data inventory:

| Firestore owner | Current localized fields | Runtime reader | Admin / seed writer | Migration owner path |
| --- | --- | --- | --- | --- |
| `app_config/public` | `supported_locales`, `default_locale`, `default_currency`, `currency_by_locale`. | `get_public_config`, `normalize_public_config`, `resolve_pricing_currency`. | `api/src/bin/seed_catalog.rs::app_config_public_document`. | `config/languages.json` as source of truth; generated/seeded public config in `M4-T01`. |
| `materials/{material_key}` | `label_i18n`, `description_i18n`, `photos[].alt_i18n`, plus non-map fields `comparison_texture_ja`, `comparison_texture_en`, `comparison_weight_ja`, `comparison_weight_en`, `comparison_usage_ja`, `comparison_usage_en`. | `/v1/catalog` resolves label, description, photo alt; material labels also feed stone-listing cards. | `api/src/bin/seed_catalog.rs::material_document`, admin material create/update/photo flows, and mock snapshots. | `api/content/i18n/catalog/materials.json`. Convert comparison preview fields to registry-keyed maps such as `comparison_i18n.texture.<lang>`, or store the equivalent under the material JSON before seeding. |
| `facet_tags/{facet_type:key}` | `label_i18n`; `aliases` are canonicalization/search helpers, not translated UI copy. | `/v1/catalog` and stone-listing filter builders resolve tag labels by locale. | `api/src/bin/seed_catalog.rs::facet_tag_document`, admin facet tag create/update flows, and mock snapshots. | `api/content/i18n/catalog/facet_tags.json`, with aliases kept locale-neutral unless a later search task adds localized aliases. |
| `stone_listings/{listing_id}` | `title_i18n`, `description_i18n`, `story_i18n`, `photos[].alt_i18n`. | `/v1/catalog`, `/v1/stone-listings`, checkout order snapshots, and app/web listing cards. | `api/src/bin/seed_catalog.rs::stone_listing_document`, admin stone-listing create/update/photo flows, and mock snapshots. | `api/content/i18n/catalog/stone_listings.json` or per-listing JSON files if the list grows. Keep `facets.*`, `material_key`, `size`, and prices locale-neutral. |
| `countries/{country_code}` | `label_i18n`. | `/v1/catalog`, order creation shipping snapshot, admin order display. | `api/src/bin/seed_catalog.rs::country_document`, admin country create/update flows, and mock snapshots. | `api/content/i18n/catalog/countries.json`. Keep ISO country code and shipping fees locale-neutral. |
| `orders/{order_id}` | `locale`, `contact.preferred_locale`, snapshot maps `material.label_i18n`, `listing.title_i18n`, `listing.description_i18n`, `listing.primary_photo.alt_i18n`, `shipping.country_label_i18n`. | Order lookup/status, admin order detail, Stripe checkout context, receipt/support language decisions. | `create_order`, `build_order_fields`, and order status/shipping admin mutations. | Runtime order snapshots remain in Firestore. Values come from catalog maps at order creation and should preserve all available locale keys for historical display. |
| `fonts/{font_key}` | `label` only; older `label_i18n` fallback is read for compatibility. | Catalog font list and seal renderer selection. | `api/src/bin/seed_catalog.rs::font_document`; admin font create/update uses single `label`. | No translation file in this plan. Font labels are product/style names and remain locale-neutral unless a later product decision changes that. |

Seed constant inventory:

| Source | Current constants | Translation ownership |
| --- | --- | --- |
| `api/src/bin/seed_catalog.rs::app_config_public_document` | Hard-coded `ja` / `en` locale list and `JPY` / `USD` currency map. | Generated from `config/languages.json` during `M4-T01`; no hand-maintained duplicate list after registry exists. |
| `api/src/bin/seed_catalog.rs::font_seeds` | 6 font records with single labels and `kanji_style` values: `zen_maru_gothic`, `kosugi_maru`, `potta_one`, `kiwi_maru`, `wdxl_lubrifont_jp_n`, `ai_generated_seal`. | Locale-neutral font catalog. Do not move to ARB/JSON unless font display names become translated product copy. |
| `api/src/bin/seed_catalog.rs::material_seeds` | 8 material master records with `label_ja`, `label_en`, `description_ja`, `description_en`: `wood`, `qingtian_stone`, `shoushan_stone`, `balin_stone`, `yili_stone`, `laos_stone`, `xixia_stone`, `frozen_stone`. | `api/content/i18n/catalog/materials.json`. Preserve Chinese-origin material names as topical content, not Chinese translations. |
| `api/src/bin/seed_catalog.rs::stone_listing_seeds` | 3 published listing records with Japanese/English title, description, story, photo alt derived from title, facet keys, size, and USD/JPY prices. | `api/content/i18n/catalog/stone_listings.json`; keep facet keys, size, listing code, Storage paths, and prices locale-neutral. |
| `api/src/bin/seed_catalog.rs::facet_tag_seeds` | 6 tag records with Japanese/English labels: `color:green`, `color:yellow`, `color:white`, `pattern:cloud`, `pattern:veined`, `pattern:plain`. | `api/content/i18n/catalog/facet_tags.json`; keep `key`, `facet_type`, and aliases canonical. |
| `api/src/bin/seed_catalog.rs::country_seeds` | 6 country records with Japanese/English labels and USD/JPY shipping fees: `JP`, `US`, `CA`, `GB`, `AU`, `SG`. | `api/content/i18n/catalog/countries.json`; fees stay in `shipping_fee_by_currency`. |
| `admin/src/main.rs::new_mock_snapshot` | Mock catalog data for admin/dev with Japanese/English maps for materials, listings, facet tags, countries, and order fixtures. | Treat as fixture data. It can either be generated from the same catalog JSON in `M5`, or kept as a small test fixture that must pass the same missing-key checks. |
| `web/src/main.rs::new_mock_catalog_source` | Web mock/dev catalog fallback used when Firestore catalog load fails in dev. | After M3/M4, derive from the same catalog JSON or mark as fixture-only so it cannot become a hidden untranslated production source. |

Checkout copy and locale propagation inventory:

| Checkout owner | Current behavior | Migration owner path |
| --- | --- | --- |
| `api/src/main.rs::build_checkout_product_name` | Hard-coded Japanese format for `ja*`: `宝石印鑑 ({listing_label}、{shape})`; all other locales use English `Stone seal ({listing_label}; {shape})`. | `api/content/i18n/checkout/<lang>.json` with keys for product name template, separator/punctuation, and shape labels. |
| `checkout_shape_label_ja` / `checkout_shape_label_en` | Hard-coded labels for `round` and `square`. Non-Japanese locales always use English. | `api/content/i18n/checkout/<lang>.json`, or a shared `shape_labels` map generated from registry-backed checkout copy. |
| `build_stripe_checkout_session_form` | Adds `lang={order_locale}` to success and cancel URLs, and uses order locale for app-return URLs. Sends `customer_email` and `payment_intent_data[receipt_email]` for Stripe-native receipts. | Keep locale propagation in code, but normalize route code through `config/languages.json`. Receipt language is not currently customized beyond Stripe/customer settings. |
| `create_order` / `validate_create_order_request` | Validates lowercase BCP-47-like `locale` and `contact.preferred_locale`, then checks both against `supported_locales`. | Registry-backed validation in `M4-T05`; route code should remain the persisted value. |
| `OrderCheckoutContext::resolve_order_listing_fields` | Resolves listing/material label from order snapshot maps using order locale and default locale. | Keep snapshot resolution; add diagnostics in `M6` when checkout falls back for release-enabled languages. |
| Gemini Kanji candidates | `reason_language` defaults to `en` if missing; API validates it as a short string and writes it back into candidate responses. Prompt labels support Japanese or English through `reason_language_label`. | Registry-backed `reason_language` mapping in `M4-T06`; not catalog copy, but it is a locale-dependent API field. |

Fallback and preservation rules for later implementation:

- `resolve_localized` currently falls back to `ja`, while this plan's general
  target fallback is registry fallback, default route code, `en`, then first
  non-empty value. M4 must intentionally update this order or document why API
  catalog fallback stays Japanese-first.
- Admin create/update flows currently edit only `ja` and `en` entries for
  `label_i18n`, `description_i18n`, `title_i18n`, `story_i18n`, and
  `alt_i18n`. Some updates preserve unknown keys by inserting into existing
  maps, but Firestore writes use full map fields. M5 must audit these writes
  before adding 68-language catalog data.
- `comparison_texture_ja/en`, `comparison_weight_ja/en`, and
  `comparison_usage_ja/en` are not `*_i18n` maps. They need a migration target
  before more languages are enabled.
- Order snapshots should not be backfilled from changed catalog translations
  after checkout. New orders should snapshot the catalog maps available at the
  time of order creation.
- Existing Chinese-related catalog values are material names and origins such
  as Qingtian, Shoushan, Balin, Yili, Xixia, Laos stone, and China production
  references. They are English/Japanese catalog content today, not `zh` or
  `zhtw` translations. When Chinese locale files are introduced, these values
  should be translated or intentionally preserved via sidecars according to the
  catalog translation policy.

#### M0-T04 Release Metadata and Deep-Link Inventory

Completed on 2026-06-18. This inventory covers app identifiers, version source,
release signing assumptions, checkout return paths, associated-domain state,
and current store-copy locations. It is the baseline needed before adding
fastlane in `M8`.

Current app identifiers and version source:

| Platform | Current value | Source file | Notes for fastlane |
| --- | --- | --- | --- |
| Android namespace | `org.finitefield.hankofield` | `app/android/app/build.gradle.kts` | Keep aligned with package name and Play Console package. |
| Android application ID | `org.finitefield.hankofield` | `app/android/app/build.gradle.kts` | Recommended Android `Appfile` package name. |
| Android app label | `STONE SIGNATURE` | `app/android/app/src/main/res/values/strings.xml` | Store metadata should use generated localized metadata, not this resource file. |
| iOS bundle identifier | `org.finitefield.hankofield` | `app/ios/Runner.xcodeproj/project.pbxproj`, surfaced through `PRODUCT_BUNDLE_IDENTIFIER` in `app/ios/Runner/Info.plist`. | Recommended iOS `Appfile` app identifier. |
| iOS display name / bundle name | `STONE SIGNATURE` | `app/ios/Runner/Info.plist` | App Store metadata should be generated separately. |
| Flutter package version | `1.1.0+11` | `app/pubspec.yaml` | Treat this as the committed version source. Local generated Android files can drift and should not be used as release metadata source. |
| Android version code/name | `flutter.versionCode` / `flutter.versionName` | `app/android/app/build.gradle.kts`, generated from Flutter tooling. | fastlane build lanes should call Flutter from `app/` or pass explicit build name/number from the release process. |
| iOS build number/name | `$(FLUTTER_BUILD_NUMBER)` / `$(FLUTTER_BUILD_NAME)` | `app/ios/Runner/Info.plist`, `app/ios/Runner.xcodeproj/project.pbxproj`. | fastlane iOS lanes should rely on Flutter build args or the committed `pubspec.yaml` version. |

Release signing and credential assumptions:

| Area | Current state | Required handling before fastlane |
| --- | --- | --- |
| Android release signing | Gradle reads `app/android/key.properties` and expects `app/android/app/upload-keystore.jks` for release builds. Release builds fail with a clear `GradleException` when either file is missing. | Treat `key.properties`, keystores, passwords, and exported AAB/APK files as private local or CI-secret material. Do not commit them. |
| Android signing fingerprint | `doc/app-release-deep-link-config.md` says `assetlinks.json` must include the release signing certificate SHA-256 fingerprint for `org.finitefield.hankofield`. | Add a fastlane/setup note for retrieving and validating the upload/app signing fingerprint before hosting association files. |
| iOS signing | Xcode project uses automatic signing style for the test target and Runner build settings attach `Runner/Runner.entitlements` to Debug, Release, and Profile. No Apple team ID or API key is committed. | fastlane should use App Store Connect API key files or CI secrets. Do not commit API key JSON/P8 files or provisioning secrets. |
| Secret ignore coverage | `.gitignore` currently ignores Firebase admin SDK files and `api/.secrets.local`, but Android signing files and future fastlane private files are not explicitly listed. Current signing files are local untracked files. | `M8-T06` must add ignore rules for Android key properties, keystores, Apple API keys, service account JSON, exported `.ipa`, generated AAB/APK/fastlane reports, and local screenshots if needed. |

Checkout return and deep-link state:

| Surface | Current route or config | Source files | Localization impact |
| --- | --- | --- | --- |
| Custom scheme | `hankofield://checkout/*` with host `checkout`. | `app/android/app/src/main/AndroidManifest.xml`, `app/ios/Runner/Info.plist`, `app/lib/features/order/domain/checkout_return.dart`. | Primary app-originated Stripe return path. Locale is carried by `lang` or `locale` query parameter. |
| Android App Links | Verified HTTPS intent filters for `finitefield.org` and `www.finitefield.org`, path prefixes `/payment`, `/en/payment`, and `/ja/payment`. | `app/android/app/src/main/AndroidManifest.xml`, covered by `app/test/platform_deep_link_config_test.dart`. | Must expand from hard-coded `en` / `ja` prefixes to registry-generated payment prefixes when release-enabled web payment paths expand. |
| iOS Universal Links | Associated domains `applinks:finitefield.org` and `applinks:www.finitefield.org`. | `app/ios/Runner/Runner.entitlements`, covered by `app/test/platform_deep_link_config_test.dart`. | Hosted AASA path list must include localized payment prefixes when they are enabled. |
| Hosted association files | No `assetlinks.json` or `apple-app-site-association` file is committed. | `doc/app-release-deep-link-config.md` documents required hosted URLs. | Add generation or documented manual hosting before broader localized payment paths are released. |
| Web payment pages | `/payment/success`, `/payment/failure`, `/ja/payment/success`, `/ja/payment/failure`; `/en/payment/*` is retained as compatibility in app link config. | `web/src/main.rs`, payment templates, and `doc/app-release-deep-link-config.md`. | M3/M10 must verify every release-enabled payment prefix. English canonical remains unprefixed, but `/en/payment/*` compatibility must not break existing app links. |
| API web return URLs | `API_PSP_STRIPE_CHECKOUT_SUCCESS_URL` and `API_PSP_STRIPE_CHECKOUT_CANCEL_URL` point to public web payment pages in `.env.prod.example`. | `.env.prod.example`, `api/src/main.rs::StripeCheckoutConfig`. | Browser checkout stays on web. The API appends checkout outcome, order ID, and `lang`. |
| API app return URLs | `API_PSP_STRIPE_APP_CHECKOUT_SUCCESS_URL=hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}` and `API_PSP_STRIPE_APP_CHECKOUT_CANCEL_URL=hankofield://checkout/cancel`. | `.env.prod.example`, `doc/app-release-deep-link-config.md`, `api/src/main.rs`. | App-originated checkout uses custom scheme when `return_to_app=true`; API appends `checkout`, `order_id`, `lang`, and `return_to=app`. |
| App return parser | Parses `hankofield://checkout/...`, `/payment/...`, `/checkout/...`, and localized web returns; strips only leading `en` or `ja` path segments today. | `app/lib/features/order/domain/checkout_return.dart`, `app/test/checkout_return_test.dart`. | Must become registry-aware before 68 localized payment routes are enabled. |

Current store-copy locations:

| Copy type | Current source | Migration target |
| --- | --- | --- |
| App name in installed Android/iOS app | `app/android/app/src/main/res/values/strings.xml`, `app/ios/Runner/Info.plist`, app localization `appTitle`. | Keep runtime app name stable unless product decides localized names. Store listing names belong in `release/store_metadata/source/<lang>.json`. |
| Flutter app description | `app/pubspec.yaml` contains a developer/package description, not store metadata. | Do not use as store listing copy. Generate store metadata from `release/store_metadata/source/<lang>.json`. |
| Web app manifest copy | `app/web/manifest.json` has PWA name, short name, and description. | Not a mobile store metadata source; include in web/PWA localization only if the web app surface is released. |
| App Store Review explanation | `doc/app-mvp-screen-design.md` has Japanese App Store Review notes explaining native app value. | Use as a reference when drafting `review_notes` / reviewer instructions, but not as localized public metadata. |
| Public legal/support URLs | App settings content and web templates point to finitefield.org privacy/contact/legal pages. | Store metadata source should carry support URL, marketing URL, and privacy policy URL per platform/locale. |
| Existing fastlane/store metadata files | No `app/android/fastlane`, `app/ios/fastlane`, `Gemfile`, or `release/store_metadata/**` files are present. | `M8` creates source JSON, generated Google Play/App Store metadata folders, Bundler files, and metadata-only lanes. |

Fastlane setup readiness notes:

- Android fastlane can use `package_name("org.finitefield.hankofield")`.
- iOS fastlane can use `app_identifier("org.finitefield.hankofield")`.
- Metadata generation must validate platform-specific locale support before
  writing Google Play or App Store folders.
- Binary upload lanes must remain separate from metadata-only lanes.
- Production lanes must require manual confirmation after internal Google Play
  and TestFlight lanes are proven.
- Deep-link association files and platform manifests must be updated from the
  language registry before release-enabled localized payment paths go live.

### M1: Registry and Read-Only Tooling

- [ ] `M1-T01` Add `config/languages.json`.
  Output: all 68 route codes with BCP-47, Flutter, text direction, fallback,
  currency, web, app, and release fields.
  Done when: `no`, `zh`, `zhtw`, and RTL entries validate correctly.
- [ ] `M1-T02` Add a registry parser shared by scripts.
  Output: one typed parser or data model used by status/check commands.
  Done when: duplicate route codes and invalid fallback values fail tests.
- [ ] `M1-T03` Add `make i18n-status`.
  Output: read-only status report for app, web, API, and release metadata.
  Done when: it reports missing files without modifying the working tree.
- [ ] `M1-T04` Add registry fixtures.
  Output: focused test fixtures for `en`, `ja`, `zh`, `zhtw`, `no`, and RTL.
  Done when: fixture tests cover route code, BCP-47, Flutter, and store fields.
- [ ] `M1-T05` Document registry update rules.
  Output: short maintainer notes for adding, disabling, or indexing a language.
  Done when: future language changes do not require reading implementation code.

### M2: Flutter App Migration

- [ ] `M2-T01` Introduce Flutter `gen-l10n`.
  Output: `app/l10n.yaml`, `app_en.arb`, `app_ja.arb`, and generated delegates.
  Done when: app builds with generated localization for English and Japanese.
- [ ] `M2-T02` Migrate existing Chinese app assets.
  Output: `app_zh.arb`, `app_zh_Hant.arb`, or approved fallback sidecars.
  Done when: existing Chinese product copy is not lost during ARB migration.
- [ ] `M2-T03` Replace hand-written localization accessors.
  Output: generated localization access in `HankoApp` and feature screens.
  Done when: compatibility wrapper is removed or documented as temporary.
- [ ] `M2-T04` Move long settings content to JSON assets.
  Output: registry-keyed JSON files under `app/assets/i18n/settings/`.
  Done when: settings content can be translated without editing Dart source.
- [ ] `M2-T05` Render language settings from the registry.
  Output: selectable language rows using `native_name`, `english_name`, and
  `app.selectable`.
  Done when: English/Japanese rows are no longer hard-coded.
- [ ] `M2-T06` Persist and normalize preferred route code.
  Output: saved `route_code` with fallback for old `en` and `ja` values.
  Done when: restart keeps the selected language and invalid values recover.
- [ ] `M2-T07` Add RTL and overflow coverage.
  Output: widget tests or screenshot checks for one RTL locale and long text.
  Done when: settings, checkout, and order screens do not visibly overflow.

### M3: Web Copy and Route Migration

- [ ] `M3-T01` Add a web registry loader.
  Output: web-side language model for route parsing, links, `hreflang`, and
  sitemap generation.
  Done when: `SUPPORTED_LOCALES` is no longer the source of truth.
- [ ] `M3-T02` Create typed web copy structs.
  Output: JSON-backed structs for common layout, SEO, top, design, about, blog,
  payment, terms, and commercial transaction pages.
  Done when: templates render data fields instead of locale conditionals.
- [ ] `M3-T03` Extract existing English, Japanese, and Chinese web copy.
  Output: localized JSON files under `web/content/i18n/`.
  Done when: current visible copy is preserved after extraction.
- [ ] `M3-T04` Replace language switcher fields.
  Output: `LanguageLink` list replacing `lang_ja_url` and `lang_en_url`.
  Done when: the switcher can render more than two languages.
- [ ] `M3-T05` Generate `hreflang`, canonical URLs, and sitemap entries.
  Output: registry-driven SEO output using `web.indexed`.
  Done when: non-indexed QA languages render but do not enter the sitemap.
- [ ] `M3-T06` Migrate blog content layout.
  Output: `web/content/blog/<slug>/<lang>.html` plus language-keyed metadata.
  Done when: English and Japanese blog pages retain their current URLs.
- [ ] `M3-T07` Add web routing tests.
  Output: tests for `/about`, `/ja/about`, `/zhtw/...`, `/en/...`, and unknown
  locale prefixes.
  Done when: unknown locale prefixes return 404 instead of English content.

### M4: API, Catalog, and Checkout Localization

- [ ] `M4-T01` Generate public config from the registry.
  Output: `/v1/config/public` locales, defaults, and currency maps are
  registry-backed.
  Done when: seed and runtime config agree on supported locales.
- [ ] `M4-T02` Move seed catalog copy to data files or map-based structures.
  Output: materials, stone listings, countries, and facet tags no longer need
  per-language Rust struct fields.
  Done when: adding `fr` does not require a new Rust field.
- [ ] `M4-T03` Preserve unknown locale keys in API writes.
  Output: merge behavior for localized Firestore maps.
  Done when: existing `fr`, `zh`, or `zhtw` keys survive updates to `ja`.
- [ ] `M4-T04` Localize checkout product labels.
  Output: data-driven checkout title and description templates.
  Done when: checkout labels support at least `en`, `ja`, `zh`, and `zhtw`.
- [ ] `M4-T05` Normalize checkout return locale handling.
  Output: consistent handling for `lang`, `locale`, and preferred locale.
  Done when: Stripe return URLs preserve the selected route code.
- [ ] `M4-T06` Define Gemini `reason_language` mapping.
  Output: registry-backed mapping from route code to prompt language.
  Done when: unsupported prompt languages fallback with visible diagnostics.
- [ ] `M4-T07` Add API tests.
  Output: tests for supported locale, missing value fallback, unsupported
  locale rejection, and checkout language persistence.
  Done when: API locale behavior can be changed safely.

### M5: Admin Data Preservation

- [ ] `M5-T01` Audit admin localized form writes.
  Output: list of every form that reads or writes a `*_i18n` map.
  Done when: high-risk overwrite paths are identified.
- [ ] `M5-T02` Add merge helpers for localized maps.
  Output: shared save behavior that edits selected keys without replacing the
  whole map.
  Done when: unknown locale keys are preserved by default.
- [ ] `M5-T03` Add preservation tests.
  Output: tests covering materials, stone listings, countries, and facet tags.
  Done when: editing Japanese values preserves `fr`, `zh`, and `zhtw` values.
- [ ] `M5-T04` Add optional compact localized-values editor.
  Output: collapsed registry-driven editor if manual admin editing is needed.
  Done when: the admin does not render 68 always-visible inputs.
- [ ] `M5-T05` Verify admin policy.
  Output: review note confirming no polling, SSE, or WebSocket behavior was
  added.
  Done when: admin remains aligned with repository policy.

### M6: Translation Workflow Tooling

- [ ] `M6-T01` Implement `make i18n-todo`.
  Output: actionable missing-key report with file, locale, key, base English
  value, fallback value, and sidecar path.
  Done when: `LANGS=` and `FILE=` filters work.
- [ ] `M6-T02` Implement `make i18n-check`.
  Output: one validation gate for registry, ARB, JSON, web copy, API content,
  sidecars, and release metadata.
  Done when: missing or malformed content returns non-zero.
- [ ] `M6-T03` Validate ARB placeholders and ICU syntax.
  Output: checks for placeholder names, metadata, plural/select syntax, and
  generated Flutter locale mapping.
  Done when: placeholder mismatch fails before runtime.
- [ ] `M6-T04` Validate JSON shape and fallback chains.
  Output: shape comparison for settings, web copy, API content, and metadata.
  Done when: fallback chains never point to missing or disabled languages.
- [ ] `M6-T05` Implement intention sidecar validation.
  Output: allowed reason code checks and per-key suppression.
  Done when: English leftovers are allowed only with approved sidecars.
- [ ] `M6-T06` Add export/import helpers.
  Output: translation handoff files that can be reviewed and imported without
  changing key order.
  Done when: generated diffs remain deterministic.
- [ ] `M6-T07` Add CI integration after checks stabilize.
  Output: CI target or documented command set for release branches.
  Done when: release-enabled languages cannot regress silently.

### M7: Pilot Language Rollout

- [ ] `M7-T01` Enable pilot languages as render-only.
  Output: registry flags for `zh`, `zhtw`, and `ar` with `web.indexed=false`
  and `release.enabled=false`.
  Done when: pilot languages render in QA without public indexing.
- [ ] `M7-T02` Fill pilot app content.
  Output: ARB and settings JSON for `zh`, `zhtw`, and `ar`.
  Done when: pilot app screens launch without fallback for primary UI.
- [ ] `M7-T03` Fill pilot web and payment content.
  Output: page JSON, payment result copy, and SEO fields for pilot routes.
  Done when: `/zh/...`, `/zhtw/...`, and `/ar/...` render correctly.
- [ ] `M7-T04` Fill pilot API and checkout content.
  Output: catalog fallback review and checkout templates for pilot languages.
  Done when: pilot checkout labels and return routes preserve locale.
- [ ] `M7-T05` Run pilot screenshot QA.
  Output: screenshots for app settings, design, checkout, web top/about/payment,
  and RTL pages.
  Done when: overflow and directionality findings are fixed or tracked.
- [ ] `M7-T06` Decide pilot public readiness.
  Output: registry flag changes for app-selectable or web-indexed status.
  Done when: each pilot language has explicit enablement evidence.

### M8: Store Metadata and fastlane

- [ ] `M8-T01` Define store metadata source schema.
  Output: `release/store_metadata/source/*.json` schema and examples for
  `en`, `ja`, `zh`, and `zhtw`.
  Done when: required fields are validated before generation.
- [ ] `M8-T02` Generate Google Play metadata.
  Output: deterministic `release/store_metadata/google_play/**` folders using
  `android_store_locale`.
  Done when: unsupported Google Play locales fail with clear messages.
- [ ] `M8-T03` Generate App Store metadata.
  Output: deterministic `release/store_metadata/app_store/**` folders using
  `ios_store_locale`.
  Done when: unsupported App Store locales fail with clear messages.
- [ ] `M8-T04` Add Android fastlane with Bundler.
  Output: `app/android/Gemfile`, `Appfile`, and metadata/internal lanes.
  Done when: metadata-only Android lane runs without uploading binaries.
- [ ] `M8-T05` Add iOS fastlane with Bundler.
  Output: `app/ios/Gemfile`, `Appfile`, and metadata/TestFlight lanes.
  Done when: metadata-only iOS lane runs without uploading binaries.
- [ ] `M8-T06` Add secret and signing guardrails.
  Output: `.gitignore` and setup notes for service accounts, Apple API keys,
  keystore files, passwords, and exported binaries.
  Done when: private release material cannot be staged accidentally.
- [ ] `M8-T07` Add screenshot metadata workflow.
  Output: screenshot naming rules and optional screengrab/deliver metadata
  preparation.
  Done when: screenshots can be matched to locale and device deterministically.

### M9: 68-Language Content Production

- [ ] `M9-T01` Create all missing locale files.
  Output: ARB, settings JSON, web JSON, API content, and metadata source
  stubs for every enabled language.
  Done when: `i18n-todo` reports missing keys instead of missing files.
- [ ] `M9-T02` Translate in script-family batches.
  Output: reviewed translations for Latin, Cyrillic, Indic, Southeast Asian,
  CJK, and RTL groups.
  Done when: each batch passes `make i18n-check LANGS=<batch>`.
- [ ] `M9-T03` Review brand, legal, and product-name holdouts.
  Output: intention sidecars for approved shared English or legal terms.
  Done when: non-English files have no unapproved English leftovers.
- [ ] `M9-T04` Run tiered layout QA.
  Output: full QA for Tier 1, screenshot QA for Tier 2, mechanical checks for
  Tier 3.
  Done when: layout issues are fixed before indexing or release enablement.
- [ ] `M9-T05` Enable language flags in stages.
  Output: separate PRs or commits for render-only, app-selectable,
  web-indexed, and store-release-enabled transitions.
  Done when: each flag change has validation evidence.
- [ ] `M9-T06` Freeze release candidate translations.
  Output: language set and metadata baseline for the next app release.
  Done when: translation changes after freeze require explicit review.

### M10: Release QA and Staged Launch

- [ ] `M10-T01` Build Android release candidate.
  Output: signed AAB build evidence.
  Done when: `flutter build appbundle --release` passes with release signing.
- [ ] `M10-T02` Build iOS release candidate.
  Output: signed IPA build evidence from a signing-capable Mac.
  Done when: `flutter build ipa --release` passes.
- [ ] `M10-T03` Verify deep links and payment return paths.
  Output: smoke-test results for custom scheme, Universal Links/App Links, and
  localized payment paths.
  Done when: checkout returns preserve route code across pilot languages.
- [ ] `M10-T04` Upload to internal Google Play track.
  Output: fastlane internal upload evidence.
  Done when: internal testers can install the build.
- [ ] `M10-T05` Upload to TestFlight.
  Output: fastlane TestFlight upload evidence.
  Done when: TestFlight testers can install the build.
- [ ] `M10-T06` Run production release signoff.
  Output: checklist covering validation gates, screenshots, metadata, rollback,
  and support readiness.
  Done when: production lane requires and records explicit manual confirmation.
- [ ] `M10-T07` Execute staged production release.
  Output: store rollout notes and monitoring checkpoints.
  Done when: rollout percentage, observed issues, and rollback decision points
  are recorded.

### M11: Post-Release Monitoring and Cleanup

- [ ] `M11-T01` Monitor locale diagnostics.
  Output: review of unsupported locale, fallback, missing content, checkout
  locale, and malformed translation logs.
  Done when: no release-enabled locale has unexpected fallback spikes.
- [ ] `M11-T02` Triage support feedback by locale.
  Output: issue list grouped by language, platform, and screen.
  Done when: translation and layout fixes have owners.
- [ ] `M11-T03` Patch high-priority translation issues.
  Output: small content-only fixes with `i18n-check` evidence.
  Done when: store-release-enabled languages remain clean after patches.
- [ ] `M11-T04` Remove temporary migration wrappers.
  Output: cleanup PR for compatibility code that is no longer needed.
  Done when: generated localization and registry paths are the only active
  localization mechanisms.
- [ ] `M11-T05` Update the release runbook.
  Output: final notes for adding future languages, store metadata updates, and
  fastlane release steps.
  Done when: the next localized release does not need rediscovery.

### Suggested PR Slices

- PR 1: `M0` baseline inventory and migration safety.
- PR 2: `M1` registry and read-only status tooling.
- PR 3: `M2` Flutter app migration.
- PR 4: `M3` web copy and route migration.
- PR 5: `M4` API, catalog, and checkout localization.
- PR 6: `M5` admin data preservation.
- PR 7: `M6` translation workflow tooling.
- PR 8: `M7` pilot language render-only QA.
- PR 9: `M8` store metadata generation and fastlane metadata lanes.
- PR 10+: `M9` language-batch translation and content production.
- PR 11: `M10` internal/TestFlight release validation.
- PR 12: `M11` post-release monitoring and cleanup.

## 13. Store Metadata Source

Add:

```text
release/store_metadata/source/en.json
release/store_metadata/source/ja.json
release/store_metadata/source/zh.json
release/store_metadata/source/zhtw.json
```

Source schema:

```json
{
  "app_name": "STONE SIGNATURE",
  "subtitle": "Custom gemstone seals",
  "short_description": "Design and order a custom gemstone seal.",
  "full_description": ["Paragraph 1", "Paragraph 2"],
  "keywords": ["hanko", "seal", "gemstone"],
  "release_notes": {
    "1.1.0": ["Localized app experience and store metadata."]
  },
  "support_url": "https://finitefield.org/contact/",
  "marketing_url": "https://finitefield.org/",
  "privacy_policy_url": "https://finitefield.org/privacy/",
  "screenshot_captions": {
    "design": "Design your seal impression",
    "stones": "Choose a one-of-a-kind stone",
    "checkout": "Order securely"
  }
}
```

Generation rules:

- Google Play folders use `android_store_locale`.
- App Store folders use `ios_store_locale`.
- Skip platform generation for null store locale values.
- Fail if `release.enabled` is true and required platform metadata is missing.
- Keep generated metadata deterministic to make diffs reviewable.

## 14. fastlane Plan

Use official fastlane concepts:

- `supply` / `upload_to_play_store` for Google Play metadata, screenshots,
  APKs, and AABs.
- `deliver` / `upload_to_app_store` for App Store Connect metadata,
  screenshots, and binaries.
- TestFlight upload via `upload_to_testflight`.
- Bundler for reproducible fastlane versions in both `app/android` and
  `app/ios`.

Run Android lanes from `app/android` and iOS lanes from `app/ios`. The
Fastfiles should use paths relative to those directories.

Recommended Android `Appfile` values:

```ruby
package_name("org.finitefield.hankofield")
```

Recommended iOS `Appfile` values:

```ruby
app_identifier("org.finitefield.hankofield")
```

Recommended Android lanes:

```ruby
default_platform(:android)

platform :android do
  lane :metadata do
    upload_to_play_store(
      skip_upload_aab: true,
      skip_upload_apk: true,
      metadata_path: "../../release/store_metadata/google_play"
    )
  end

  lane :internal do
    sh("cd .. && flutter build appbundle --release")
    upload_to_play_store(
      track: "internal",
      aab: "../build/app/outputs/bundle/release/app-release.aab"
    )
  end
end
```

Recommended iOS lanes:

```ruby
default_platform(:ios)

platform :ios do
  lane :metadata do
    deliver(
      metadata_path: "../../release/store_metadata/app_store",
      skip_binary_upload: true,
      skip_screenshots: true
    )
  end

  lane :testflight do
    sh("cd .. && flutter build ipa --release")
    upload_to_testflight
  end
end
```

Credential rules:

- Google Play service account data must come from local ignored files or CI
  secrets.
- Apple credentials must come from App Store Connect API key files or CI
  secrets, not committed files.
- Android keystore and passwords must be provided by local files or CI secrets.
- CI should reconstruct private signing files at build time.
- Do not echo secret values in scripts.

## 15. QA Strategy

### Tier 1: Full QA Languages

Use full manual and screenshot QA:

- `en`
- `ja`
- `zh`
- `zhtw`
- `ar`

Coverage:

- app onboarding
- design flow
- kanji candidates
- seal style selection
- seal generation result
- stone list and detail
- order flow
- payment result
- settings language selection
- order lookup
- web top/design/about/blog/payment/legal pages

### Tier 2: Script and Layout Coverage

Use targeted screenshots:

- `de` for long Latin text
- `fr` or `es` for punctuation and common store copy
- `ru` or `uk` for Cyrillic
- `hi` for Devanagari
- `ta` or `te` for South Indian scripts
- `th` for Thai wrapping
- `ko` for Korean
- `he` or `ur` for RTL

### Tier 3: Mechanical Coverage

All remaining languages:

- must pass `make i18n-check`
- must build in app/web contexts if enabled
- must have store metadata when release-enabled

## 16. Test Plan

### Unit Tests

App:

- registry route code normalization
- saved locale fallback
- generated localization availability
- settings JSON loader fallback
- `reasonLanguage` mapping

Web:

- path locale parsing
- canonical URL generation
- language link generation
- `hreflang` generation
- sitemap inclusion/exclusion
- copy JSON deserialization

API:

- public config normalization
- locale support checks
- catalog fallback order
- checkout product template resolution
- order locale persistence
- `reason_language` validation and fallback

Admin:

- localized map preservation on save
- no loss of unknown locale keys

### Integration Tests

- App requests `/v1/catalog?locale=zh`.
- App creates order with `locale=zhtw` and `preferred_locale=zhtw`.
- API creates checkout session preserving `lang=zhtw`.
- Web payment result renders for `/zhtw/payment/success`.
- Unknown `/xx/about` returns 404.
- `/en/about` redirects to or canonicalizes as `/about` if compatibility route
  is retained.

### Manual Tests

- Switch language in app settings.
- Restart app and verify selected locale persists.
- Open web language switcher and move between localized pages.
- Confirm RTL layout in app and web.
- Confirm Stripe return routes include and preserve `lang`.
- Confirm Universal Link/App Link behavior after route expansion.

### Release Tests

- `flutter build appbundle --release`
- `flutter build ipa --release`
- metadata-only Android fastlane lane
- metadata-only iOS fastlane lane
- Google Play internal upload
- TestFlight upload

## 17. Rollout Plan

### Phase 1: Foundation Only

- Add registry and read-only status tooling.
- No user-visible changes.

Rollback:

- Revert registry/tooling commit.

### Phase 2: English/Japanese Migration

- Move existing `en` and `ja` strings to new file layout.
- Preserve behavior.
- Keep only `en` and `ja` selectable.

Rollback:

- Revert localization migration.
- Keep existing app/web behavior.

### Phase 3: Pilot Languages

- Enable `zh`, `zhtw`, and `ar` for render-only QA.
- Do not index web pages or release store metadata yet.

Rollback:

- Set registry flags `web.indexed=false`, `app.selectable=false`,
  `release.enabled=false`.

### Phase 4: Web Indexing

- Index languages only after page title, meta description, body copy, and
  `hreflang` validation pass.

Rollback:

- Set `web.indexed=false` for affected language.
- Regenerate sitemap.

### Phase 5: App Store Metadata

- Generate and upload metadata-only updates.
- Keep binary release lanes separate.

Rollback:

- Restore previous store metadata through fastlane or store console.

### Phase 6: Binary Release

- Use internal Google Play and TestFlight first.
- Promote to production only after manual signoff.

Rollback:

- Use store rollback/promote previous build where supported.
- Disable problematic locale in remote/public config when possible.
- Ship hotfix with language hidden if client-side issue is severe.

## 18. Monitoring and Diagnostics

Add logs/diagnostics for:

- unsupported locale requested
- fallback locale used
- missing content file
- malformed translation file
- checkout locale and preferred locale
- store metadata generation skipped because platform locale is missing

Do not log private customer data beyond existing policy. Locale, route code, and
fallback reason are safe diagnostic fields.

## 19. Security and Privacy

- Translation files must not contain secrets.
- Store metadata source must not contain credentials.
- fastlane service account JSON and Apple API keys must be ignored or stored in
  CI secrets.
- Local Android signing files and passwords must not be staged.
- Translation tooling must not upload customer/order data to external services.
- If external translation services are used later, only source strings and
  non-private product copy may be exported.

## 20. Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| UI overflow | Broken app/web screens | Tiered screenshots, flexible layout, no single-line assumptions. |
| Font coverage gaps | Missing glyphs | Add UI font fallback; keep seal rendering font rules separate. |
| RTL layout regressions | Incorrect navigation and forms | Registry-driven `dir`, RTL screenshots, review directional icons. |
| Store locale mismatch | fastlane upload failure | Store platform locale mapping in registry and validate before upload. |
| Legal copy divergence | Legal/review risk | Japanese governing text, manual legal review, sidecar reasons for legal names. |
| Credential exposure | Security incident | Ignore private files, use CI secrets, avoid logging secrets. |
| Silent fallback | Users see wrong language | Fallback diagnostics and release-enabled fallback checks. |
| Admin overwrites translations | Data loss | Merge map writes and add preservation tests. |
| Deep link route gaps | Checkout return failure | Update association files and smoke-test all payment path variants. |

## 21. Validation Gates

Before enabling a language for app selection:

- ARB exists and passes `i18n-check`.
- Long settings JSON exists or has approved fallback.
- App launches in the locale.
- Settings screen can select and persist the locale.
- Tier-appropriate screenshot QA passes.

Before indexing a web language:

- Page copy exists for all indexed routes.
- SEO title and meta description are translated.
- `hreflang` output includes the language.
- Sitemap includes the language.
- Unknown locale and fallback behavior are tested.

Before enabling a release language:

- Store metadata source exists.
- Platform-specific metadata folders generate.
- Unsupported platform locale mappings are explicit.
- Screenshots are ready or intentionally skipped.
- Metadata-only fastlane lane passes.

Before production binary release:

- `flutter test` passes.
- Rust API tests pass.
- Rust web tests pass.
- `make i18n-check` passes for all release-enabled languages.
- `flutter build appbundle --release` passes.
- `flutter build ipa --release` passes on a signing-capable Mac.
- Google Play internal lane succeeds.
- TestFlight lane succeeds.
- Manual release signoff is recorded.

## 22. Open Assumptions

- The 68 finitefield.org route codes are the desired product language set for
  Stone Signature.
- English remains the canonical unprefixed web URL.
- Japanese remains `JPY`; other locales use `USD` until pricing policy changes.
- Admin can remain Japanese/internal for the initial multilingual rollout.
- Store locale availability will be validated during fastlane metadata setup,
  because Google Play and App Store Connect accepted locale lists can differ.
- CI provider and secret-storage mechanism are not chosen yet.

## 23. References

- Flutter internationalization:
  https://docs.flutter.dev/ui/internationalization
- Flutter continuous delivery:
  https://docs.flutter.dev/deployment/cd
- fastlane supply:
  https://docs.fastlane.tools/actions/supply/
- fastlane deliver:
  https://docs.fastlane.tools/actions/deliver/
- fastlane screengrab:
  https://docs.fastlane.tools/actions/screengrab/
