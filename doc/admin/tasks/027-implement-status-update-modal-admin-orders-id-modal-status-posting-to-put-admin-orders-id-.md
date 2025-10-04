# Implement status update modal (`/admin/orders/{id}/modal/status`) posting to `PUT /admin/orders/{id}:status` and updating the UI inline.

**Parent Section:** 5. Orders & Operations > 5.1 Orders List & Detail
**Task ID:** 027

## Goal
Provide modal to update order status via `PUT /admin/orders/{id}:status`.

## Implementation Steps
1. Modal GET handler renders form with current status, allowed next statuses, note textarea.
2. Submit via htmx with CSRF header; on success trigger toast and partial refresh of status cell + timeline.
3. Validate transitions client-side (disable invalid options) and server-side (display error message).
4. Log action to audit service.
