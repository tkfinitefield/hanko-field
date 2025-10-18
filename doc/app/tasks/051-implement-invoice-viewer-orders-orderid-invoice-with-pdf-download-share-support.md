# Implement invoice viewer (`/orders/:orderId/invoice`) with PDF download/share support.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 051

## Goal
Provide invoice viewer with download support.

## Implementation Steps
1. Fetch invoice metadata and secure download link.
2. Render summary details; use native PDF viewer where available.
3. Handle pending invoice state with refresh option.

## Material Design 3 Components
- **App bar:** `Small top app bar` with share `Icon button`.
- **Preview area:** `Surface` embedding PDF renderer framed by `Outlined card` boundary.
- **Metadata chips:** `Assist chips` for tax status and invoice state.
- **Actions:** `Filled button` for download and `Text button` for send via email.
