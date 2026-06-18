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

## 11. Implementation Plan by Subsystem

### Milestone 1: Foundation Registry

Files likely touched:

- `config/languages.json`
- `Makefile`
- `scripts/i18n/*`
- `doc/multilingual-release-plan.md`

Tasks:

- [ ] Add `config/languages.json` with all 68 route codes.
- [ ] Mark `web.enabled`, `app.enabled`, and `release.enabled` separately.
- [ ] Set `release.enabled` false for all newly added languages until store
  metadata and screenshots are ready.
- [ ] Implement registry validation.
- [ ] Add `make i18n-status`.
- [ ] Add tests or snapshot fixtures for `zh`, `zhtw`, `no`, and RTL entries.

Acceptance criteria:

- `make i18n-status` lists all 68 route codes.
- Duplicate route codes fail validation.
- Invalid fallback codes fail validation.
- `no` remains a string route code.
- `zhtw` resolves to Traditional Chinese platform metadata.

### Milestone 2: Flutter Generated Localization

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

Tasks:

- [ ] Create base `app_en.arb` from existing English strings.
- [ ] Create `app_ja.arb` from existing Japanese strings.
- [ ] Configure Flutter `gen-l10n`.
- [ ] Replace hand-written localization accessors with generated accessors.
- [ ] Keep a temporary compatibility wrapper only if it materially reduces the
  migration risk.
- [ ] Move long settings content to JSON assets.
- [ ] Load language settings rows from the registry.
- [ ] Store preferred `route_code`.
- [ ] Add fallback for old saved `en` and `ja` values.
- [ ] Add RTL handling.
- [ ] Add UI tests for `en`, `ja`, `zh`, `zhtw`, and one RTL locale once those
  files exist.

Acceptance criteria:

- `flutter gen-l10n` or `flutter pub get` generates localization output.
- `flutter test` passes.
- English and Japanese app text remains equivalent to current behavior.
- Language settings no longer contains hard-coded English/Japanese rows.
- Missing settings JSON shows a recoverable error state or falls back without
  crashing.

### Milestone 3: Web Copy Extraction

Files likely touched:

- `web/src/main.rs`
- `web/templates/*.html`
- `web/content/i18n/**/*.json`
- `web/blog/articles/*`
- `web/content/blog/**/*`
- `web/Makefile`

Tasks:

- [ ] Add language registry loader for `web`.
- [ ] Replace `SUPPORTED_LOCALES` with registry-backed validation.
- [ ] Introduce `LanguageLink` and page copy structs.
- [ ] Extract `top`, `index/design`, `about`, `blog_index`, `payment_success`,
  `payment_failure`, `terms`, and `commercial_transactions` copy to JSON.
- [ ] Remove `lang_ja_url` and `lang_en_url` fields from templates.
- [ ] Replace hard-coded `hreflang` tags with a loop over language links.
- [ ] Generate sitemap entries from indexed registry languages.
- [ ] Migrate blog article metadata and bodies to language-keyed content.
- [ ] Preserve current English and Japanese URLs.
- [ ] Keep `/en/...` compatibility behavior if existing inbound links require
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

### Milestone 4: API, Firestore Seed, and Checkout

Files likely touched:

- `api/src/main.rs`
- `api/src/bin/seed_catalog.rs`
- `api/content/i18n/**/*`
- `doc/firebase-firestore-design.md`
- `doc/app-release-deep-link-config.md`

Tasks:

- [ ] Load or generate public config supported locales from the registry.
- [ ] Replace hard-coded seed locale arrays with registry data.
- [ ] Move catalog seed text from Rust structs to data files or map-based seed
  structures.
- [ ] Preserve unknown locale keys when reading/writing Firestore maps.
- [ ] Extend checkout product labels to data-driven localized templates.
- [ ] Define `reason_language` mapping for Gemini prompts:
  - supported languages use their BCP-47 code if prompt quality is acceptable
  - unsupported prompt languages fallback to English
  - fallback must be visible in response diagnostics
- [ ] Add tests for `zh`, `zhtw`, unsupported locale rejection, and fallback.

Acceptance criteria:

- `/v1/config/public` returns registry-driven supported locales.
- Catalog requests for a supported locale do not fail.
- Missing catalog values fallback predictably.
- Checkout URLs preserve selected `lang`.
- Checkout product names are data-driven for at least `en`, `ja`, `zh`, and
  `zhtw` before wider rollout.

### Milestone 5: Admin Data Preservation

Files likely touched:

- `admin/src/main.rs`
- `admin/templates/material_*`
- `admin/templates/stone_listing_*`
- `admin/templates/country_*`
- `admin/templates/facet_tag_*`

Tasks:

- [ ] Audit every admin form that writes a Firestore `*_i18n` map.
- [ ] Ensure saves merge edited fields into existing maps instead of replacing
  maps with only `ja` / `en`.
- [ ] Add tests for preserving unknown locale keys.
- [ ] Optionally add a collapsed localized-values editor for catalog records.

Acceptance criteria:

- Editing a Japanese material label preserves existing `fr`, `zh`, and `zhtw`
  values.
- Admin remains usable without rendering 68 visible inputs by default.
- No admin polling, SSE, or WebSocket behavior is added.

### Milestone 6: Translation Tooling

Files likely touched:

- `Makefile`
- `scripts/i18n/*`
- `app/lib/l10n/*`
- `app/assets/i18n/**/*`
- `web/content/i18n/**/*`
- `api/content/i18n/**/*`
- `release/store_metadata/source/*`

Tasks:

- [ ] Implement `make i18n-todo`.
- [ ] Implement `make i18n-check`.
- [ ] Implement sidecar validation.
- [ ] Implement placeholder/ICU validation for ARB.
- [ ] Implement JSON shape validation for long-form content.
- [ ] Implement English-leftover checks.
- [ ] Add CI target once checks are stable.

Acceptance criteria:

- Missing locale files are reported with actionable paths.
- Placeholder mismatch fails the check.
- Unapproved English leftovers fail the check for non-English locales.
- Approved sidecar entries suppress only the intended key.
- `LANGS=` and `FILE=` filters work.

### Milestone 7: 68-Language Content Rollout

Files likely touched:

- all localization content paths
- release metadata source paths

Tasks:

- [ ] Translate base app ARB files.
- [ ] Translate app long-form settings files.
- [ ] Translate web page copy.
- [ ] Translate blog metadata and bodies for indexed languages.
- [ ] Translate catalog and checkout seed content.
- [ ] Translate store metadata.
- [ ] Run tiered QA.
- [ ] Enable languages in stages:
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

### Milestone 8: Store Metadata and fastlane

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

Tasks:

- [ ] Add source store metadata JSON per release-enabled language.
- [ ] Generate Google Play metadata folders.
- [ ] Generate App Store metadata folders.
- [ ] Add fastlane with Bundler.
- [ ] Add metadata-only Android lane.
- [ ] Add metadata-only iOS lane.
- [ ] Add Google Play internal lane.
- [ ] Add TestFlight lane.
- [ ] Add production lanes only after internal lanes are proven.
- [ ] Add `.gitignore` entries for private service account JSON, Apple API key
  files, exported `.ipa`, generated keystores, and local fastlane reports if
  needed.

Acceptance criteria:

- `bundle exec fastlane metadata` can run from `app/android` and `app/ios`
  without uploading binaries.
- Internal Google Play lane uploads an AAB to internal testing.
- TestFlight lane uploads a signed IPA.
- Production lanes require explicit manual confirmation.
- Store metadata generation validates unsupported store locales before upload.

## 12. Store Metadata Source

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

## 13. fastlane Plan

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

## 14. QA Strategy

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

## 15. Test Plan

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

## 16. Rollout Plan

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

## 17. Monitoring and Diagnostics

Add logs/diagnostics for:

- unsupported locale requested
- fallback locale used
- missing content file
- malformed translation file
- checkout locale and preferred locale
- store metadata generation skipped because platform locale is missing

Do not log private customer data beyond existing policy. Locale, route code, and
fallback reason are safe diagnostic fields.

## 18. Security and Privacy

- Translation files must not contain secrets.
- Store metadata source must not contain credentials.
- fastlane service account JSON and Apple API keys must be ignored or stored in
  CI secrets.
- Local Android signing files and passwords must not be staged.
- Translation tooling must not upload customer/order data to external services.
- If external translation services are used later, only source strings and
  non-private product copy may be exported.

## 19. Risks and Mitigations

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

## 20. Validation Gates

Before enabling a language for app selection:

- [ ] ARB exists and passes `i18n-check`.
- [ ] Long settings JSON exists or has approved fallback.
- [ ] App launches in the locale.
- [ ] Settings screen can select and persist the locale.
- [ ] Tier-appropriate screenshot QA passes.

Before indexing a web language:

- [ ] Page copy exists for all indexed routes.
- [ ] SEO title and meta description are translated.
- [ ] `hreflang` output includes the language.
- [ ] Sitemap includes the language.
- [ ] Unknown locale and fallback behavior are tested.

Before enabling a release language:

- [ ] Store metadata source exists.
- [ ] Platform-specific metadata folders generate.
- [ ] Unsupported platform locale mappings are explicit.
- [ ] Screenshots are ready or intentionally skipped.
- [ ] Metadata-only fastlane lane passes.

Before production binary release:

- [ ] `flutter test` passes.
- [ ] Rust API tests pass.
- [ ] Rust web tests pass.
- [ ] `make i18n-check` passes for all release-enabled languages.
- [ ] `flutter build appbundle --release` passes.
- [ ] `flutter build ipa --release` passes on a signing-capable Mac.
- [ ] Google Play internal lane succeeds.
- [ ] TestFlight lane succeeds.
- [ ] Manual release signoff is recorded.

## 21. Open Assumptions

- The 68 finitefield.org route codes are the desired product language set for
  Stone Signature.
- English remains the canonical unprefixed web URL.
- Japanese remains `JPY`; other locales use `USD` until pricing policy changes.
- Admin can remain Japanese/internal for the initial multilingual rollout.
- Store locale availability will be validated during fastlane metadata setup,
  because Google Play and App Store Connect accepted locale lists can differ.
- CI provider and secret-storage mechanism are not chosen yet.

## 22. References

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
