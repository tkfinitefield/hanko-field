# Implement invoice request modal (`/admin/orders/{id}/modal/invoice`) calling `POST /admin/invoices:issue` and reflecting result in tabs.

**Parent Section:** 5. Orders & Operations > 5.1 Orders List & Detail
**Task ID:** 029

## Goal
Enable staff to issue invoice via `POST /admin/invoices:issue`.

## Implementation Steps
1. Modal includes invoice template selection, recipient email, notes.
2. Submit to backend; upon success refresh invoice tab and show download link.
3. If async job, display job ID and poll for completion.
