# Implement design share/download functionality with signed URL workflow.

**Parent Section:** 4. Design Creation Flow
**Task ID:** 028

## Goal
Implement design sharing/downloading flow.

## Implementation Steps
1. Request signed download URLs via API.
2. Provide UI for share link generation and copy.
3. Track analytics for downloads and shares.

## UI Components
- **Layout:** `ModalLayout` triggered from editor, with `ModalHeader` showing design name.
- **Preview:** `SharePreview` card depicting watermark options.
- **Controls:** `FormSection` with format `Select`, size `RadioGroup`, watermark `Switch`, expiry `DatePicker`.
- **Link panel:** `LinkCard` showing generated URL, copy `IconButton`, embed snippet.
- **Status alerts:** `InlineAlert` for expiration, permission warnings.
- **CTA footer:** `ModalFooter` with `PrimaryButton` for create link and `SecondaryButton` to close.
