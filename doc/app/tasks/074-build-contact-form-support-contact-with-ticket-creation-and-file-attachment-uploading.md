# Build contact form (`/support/contact`) with ticket creation and file attachment uploading.

**Parent Section:** 12. Support & Status
**Task ID:** 074

## Goal
Implement contact form submitting support tickets.

## Implementation Steps
1. Provide form fields (topic, message, attachments) with validation.
2. Handle file upload using backend signed URLs.
3. Show submission confirmation and track ticket ID.

## Material Design 3 Components
- **App bar:** `Small top app bar` with history `Icon button`.
- **Form:** `Outlined text fields` for subject, message, order ID plus `Assist chips` for quick templates.
- **Attachment row:** `List item` with `Icon button` trigger for file picker.
- **Actions:** `Filled tonal button` to submit and `Text button` to cancel.
