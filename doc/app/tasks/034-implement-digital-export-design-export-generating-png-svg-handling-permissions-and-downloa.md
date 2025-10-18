# Implement digital export (`/design/export`) generating PNG/SVG, handling permissions and download/share sheets.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 034

## Goal
Generate digital assets for download/share.

## Implementation Steps
1. Render high-res PNG/SVG, ensuring color profile alignment.
2. Request storage permissions; offer share sheets and file save location selection.
3. Optionally apply watermark for social sharing.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with download history `Icon button`.
- **Format selector:** `Segmented buttons` switching between PNG, SVG, and PDF.
- **Options:** `List items` with `Switches` for transparent background, bleed, metadata.
- **CTA area:** `Filled button` for export and `Outlined button` for share sheet.
