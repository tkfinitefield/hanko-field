# Build order detail (`/account/orders/{orderId}`) with tabbed fragments for summary, payments, production, tracking, invoice.

**Parent Section:** 6. Account & Library
**Task ID:** 038

## Goal
Build security page for linked accounts/2FA.

## Implementation Steps
1. Display linked providers with status; provide link/unlink modals.
2. Surface 2FA status and setup instructions.
3. Integrate with backend for token revocation.

## UI Components
- **Layout:** `AccountLayout` with breadcrumb and order number `SectionHeader`.
- **Status header:** `StatusPanel` with current state, timeline `Steps`, contact support button.
- **Tabs:** `ContentTabs` for Summary, Payments, Production, Tracking, Invoice with fragments.
- **Content cards:** `DetailCard` for addresses, payment, items with reorder `Button`.
- **Timeline:** `Timeline` component for production/shipping events.
- **Document drawer:** `DocumentDrawer` for invoices/labels downloads.
