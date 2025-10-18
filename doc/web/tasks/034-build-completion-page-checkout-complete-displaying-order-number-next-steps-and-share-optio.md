# Build completion page (`/checkout/complete`) displaying order number, next steps, and share options.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 034

## Goal
Display order completion page.

## Implementation Steps
1. Show order number, summary, next actions (tracking, invoice).
2. Provide share links and support contact info.
3. Clear cart data in session.

## UI Components
- **Layout:** `CheckoutLayout` variant with celebratory `HeroCard`.
- **Order confirmation:** `ConfirmationPanel` showing order number, status `Badge`, download receipt.
- **Next steps:** `TaskList` cards for track shipment, download designs, refer friend.
- **Share strip:** `SocialShareBar` with icons and referral copy.
- **Recommendations:** `ProductRail` for accessories and `GuideCard` for onboarding docs.
- **Support callout:** `SupportBanner` linking to help center.
