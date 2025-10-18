# Implement design type selection page (`/design/new`) with CTA routing to editor.

**Parent Section:** 4. Design Creation Flow
**Task ID:** 022

## Goal
Implement design type selection page.

## Implementation Steps
1. Render cards for text input, image upload, logo engraving with descriptions.
2. Ensure responsive layout and analytics tracking for selection.
3. Navigate to `/design/editor` with chosen mode parameter.

## UI Components
- **Layout:** `SiteLayout` with creative `HeroSection` introducing design flows.
- **Option grid:** `SelectionGrid` of `OptionCard` entries (Text, Upload, Logo) with iconography.
- **Filters:** `FilterChips` for use case/industry to adjust recommendations.
- **Feature highlights:** `FeatureList` column explaining AI assist, templates, upload support.
- **CTA bar:** Bottom `StickyActionBar` containing primary `PrimaryButton` and secondary link.
- **Help banner:** `InlineHelp` linking to tutorial guide.
