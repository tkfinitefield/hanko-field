# Support asset uploads (preview images, SVGs) via integration with assets signed URL workflow inside modals.

**Parent Section:** 6. Catalog Management
**Task ID:** 040

## Goal
Support uploading preview images/SVGs within catalog modals.

## Implementation Steps
1. Integrate assets service to request signed upload URLs before form submission.
2. Provide UI component to upload file (async) and store asset ID hidden input.
3. Show thumbnail preview and allow replace/remove.
4. Validate file type/size client-side.
