# Storage Layout

## Buckets

| Bucket | Purpose | Retention | Encryption | IAM Highlights |
| --- | --- | --- | --- | --- |
| `gs://hanko-field-$ENV-assets` | Design uploads, rendered previews, invoice PDFs, fonts. | 365d default; previews auto-expire after 180d via lifecycle rule. | CMEK (`projects/hanko-sec/locations/asia-northeast1/keyRings/core/cryptoKeys/app-data`) | App service (Cloud Run) RW, Admin UI RW, AI worker RO, Signed URL (upload/download) scoped per object. |
| `gs://hanko-field-$ENV-logs` | Structured logs export (optional for retention) | 540d | Default Google-managed | Security team RO, Ops RO. |
| `gs://hanko-field-$ENV-exports` | Scheduled Firestore exports, BigQuery sync temp files | 30d | CMEK same as assets | DevOps RW, Data Eng RW. |

> `$ENV` = `dev`, `stg`, `prod`. Buckets created via Terraform with uniform bucket-level access.

## Object Layout

### `gs://hanko-field-$ENV-assets`

```
assets/
  designs/
    {designId}/
      sources/{uploadId}/{filename}
      previews/{versionId}/{size}.png
      ai/{suggestionId}/{filename}
  orders/
    {orderId}/
      proofs/{timestamp}.pdf
      invoices/{invoiceNumber}.pdf
  content/
    guides/{guideId}/{hash}.md
    pages/{pageId}/{hash}.html
  fonts/
    {fontId}/files/{filename}
  temp/
    {uuid}/... (24h TTL via lifecycle rule)
```

- Previews and proofs are immutable; version folders prevent cache races.
- `temp/` objects tagged with `temp=true` and deleted after 24h.
- Fonts restricted to admin and rendering service via IAM condition.

### Metadata & Tags

- Objects tagged with `pii=true` for deliveries containing addresses (orders/).
- Custom metadata:
  - `x-hanko-origin` (api|admin|worker) for traceability.
  - `x-hanko-retain-until` ISO date for lifecycle overrides.

## Access Patterns

- Signed uploads scoped to `assets/designs/{designId}/sources/{uploadId}/*` with 15m expiry.
- Signed downloads restricted to previews, proofs, or invoices belonging to the requesting user.
- Admin UI uses service account with conditional IAM (`resource.name.startsWith("projects/_/buckets/.../objects/assets/content/")`).

## Compliance

- CMEK usage logged via Cloud KMS audit logs; rotation every 180d.
- Enable Object Versioning in `prod` only for `orders/*` path to support evidence retention.
- `pii=true` tagged objects mirrored weekly to secure archive bucket managed by Ops.
