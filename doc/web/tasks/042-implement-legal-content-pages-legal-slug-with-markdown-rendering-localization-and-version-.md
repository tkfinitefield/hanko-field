# Implement legal content pages (`/legal/{slug}`) with markdown rendering, localization, and version tracking.

**Parent Section:** 7. Support & Legal
**Task ID:** 042

## Goal
Render legal content pages with localization and version tracking.

## Implementation Steps
1. Fetch content (markdown/HTML) from CMS or static files; sanitize output.
2. Display revision date and provide PDF download if required.
3. Support locale toggles and canonical links.

## UI Components
- **Layout:** `StaticLayout` with legal `SectionHeader` and locale switch.
- **Localization controls:** `LocaleSwitcher` toggling languages (htmx reload).
- **Document body:** `MarkdownRenderer` with `Prose` styling, links, headings.
- **Version table:** `VersionTable` listing history with download `LinkButton`.
- **Feedback widget:** `WasHelpful` inline feedback form with `RadioGroup`.
- **Footer:** `LegalFooter` referencing compliance contacts.
