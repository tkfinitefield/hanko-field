# Build add-ons screen (`/products/:productId/addons`) for optional accessories with upsell logic.

**Parent Section:** 6. Shop & Product Browsing
**Task ID:** 040

## Goal
Allow selection of optional accessories.

## Implementation Steps
1. Fetch add-ons for selected product; group by type (case, box, ink).
2. Display toggle or checkbox list updating summary price.
3. Update cart entry with selected add-ons.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with clear all `Text button` action.
- **Add-on list:** `Two-line list items` with thumbnails and trailing `Switch` for selection.
- **Upsell banner:** `Outlined card` highlighting recommended bundle.
- **Footer actions:** `Filled tonal button` to continue and `Outlined button` to skip.
