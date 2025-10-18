# Build support page (`/support`) with contact form, FAQ links, and response handling.

**Parent Section:** 7. Support & Legal
**Task ID:** 041

## Goal
Implement support page with contact form and FAQ links.

## Implementation Steps
1. Build contact form posting to backend support endpoint with validation.
2. Provide FAQ quick links and top questions list.
3. Display confirmation and error handling.

## UI Components
- **Layout:** `SiteLayout` with support `SectionHeader` and breadcrumb.
- **Contact form:** `SupportForm` with `Input`, `Select`, `Textarea`, file upload `Dropzone`.
- **FAQ accordion:** `FAQAccordion` for top issues with expand/collapse.
- **Channel cards:** `SupportChannelCard` for chat, email, phone.
- **Response timeline:** `ResponseTimeline` listing ticket SLAs and status.
- **CTA band:** `CalloutBanner` for community/forum links.
