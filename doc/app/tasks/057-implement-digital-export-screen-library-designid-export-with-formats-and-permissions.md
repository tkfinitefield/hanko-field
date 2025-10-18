# Implement digital export screen (`/library/:designId/export`) with formats and permissions.

**Parent Section:** 9. My Hanko Library
**Task ID:** 057

## Goal
Provide digital export options from library.

## Implementation Steps
1. Offer format selection (PNG/SVG/PDF) and scaling choices.
2. Handle permissions and sharing similar to design export screen.

## Material Design 3 Components
- **App bar:** `Small top app bar` with history `Icon button`.
- **Format selection:** `Segmented buttons` for file type with `Assist chips` indicating recommended use.
- **Permissions:** `List items` hosting `Switches` for watermark, expiry, download rights.
- **CTA:** `Filled tonal button` for generate link and `Outlined button` for revoke all.
