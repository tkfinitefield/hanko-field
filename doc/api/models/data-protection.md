# Data Protection & Redaction

## Classification Levels

| Level | Description | Examples | Logging Policy |
| --- | --- | --- | --- |
| P0 | Public metadata, safe for caching | template names, product shape, font preview URLs | Log freely, ensure no user association. |
| P1 | Internal operational data without PII | production queue assignments, stock reservations | Log with request ID; redact when combined with user identity. |
| P2 | User-identifiable or contact info | user.displayName, addresses, phone, email, payment last4 | Mask in logs (replace with `***`), store encrypted at rest (Firestore default + CMEK for Storage). |
| P3 | Sensitive financial/compliance data | promotion usage counts per user, payment failure codes | Restrict to service logs, never emit in client-visible payloads, enable Access Transparency. |

## Redaction Rules

- **Firestore Triggers / API Logs**: Use structured logging keys `userRef`, `orderId` only; avoid dumping entire document payloads. Mask contact fields using helper (`mask.PII()`).
- **Audit Logs Collection**: Persist redacted actors (`displayNameMasked`) and subject references only. Raw diff stored in Storage `logs` bucket with 30-day retention.
- **Crash/Error Reporting**: Attach references (`orderRef`) instead of full documents. When including validation errors, replace user inputs with hashed tokens.

## Field Handling Checklist

- `users.email`, `users.phone`, `orders.shippingAddress.*` → mark with `piiMasked=true` when redaction applied. Staff deactivation flow overwrites address fields with `***`.
- `payments.raw` payload trimmed to PSP IDs, timestamps, status reason. webhook handler stores original JSON encrypted in `logs` bucket (P3).
- `assets` metadata should never include user-entered free text; store descriptive text in Firestore where masking strategy exists.
- Enable Firestore TTL on `stockReservations.expiresAt`, `aiJobs.expiresAt` to auto-delete ephemeral data.
- Use IAM Conditions to ensure only admin service accounts can read P2/P3 fields (`roles/firestore.user` not granted to frontends).

## Data Residency & Backups

- Primary region: `asia-northeast1`. Nightly export to `gs://hanko-field-$ENV-exports` (CMEK protected).
- Long-term archive of P2/P3 resides in `hanko-field-prod-archive` (separate project with VPC-SC). Access requires break-glass procedure.
- Backups retain for 35 days rolling; invoices & audit logs mirrored for 7 years to satisfy compliance.
