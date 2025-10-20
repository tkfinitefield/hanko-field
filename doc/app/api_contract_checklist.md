# API Contract Checklist (Mobile ↔ Backend)

Scope: Align Flutter mobile payloads and flows with backend endpoints defined in `doc/api/api_design.md` and model schemas under `doc/api/models/`.

## Global Conventions
- [ ] Base URL uses `/api/v1` (see doc/api/api_design.md).
- [ ] Auth via Firebase ID Token in `Authorization: Bearer <token>` for user endpoints.
- [ ] Idempotency-Key header present on POST/PUT/PATCH/DELETE.
- [ ] Pagination: `pageSize`/`pageToken` request; `nextPageToken` response.
- [ ] Sorting/filters follow `orderBy` and `filter` query patterns when supported.
- [ ] Money fields use minor units (integers); currency is ISO 4217 string.
- [ ] Timestamps are ISO-8601 UTC (e.g., `2025-10-04T12:00:00Z`).
- [ ] Error envelope and HTTP codes documented; UI maps known `error.code` to user-friendly messages.

## Profile & Account
Models: `doc/api/models/users.schema.yaml`, `users.address.schema.yaml`, `users.paymentMethod.schema.yaml`
- [ ] GET/PUT `/me` fields (displayName, locale, etc.) mapped; immutable fields guarded.
- [ ] Addresses CRUD under `/me/addresses` mapped to form models.
- [ ] Payment methods list references PSP tokens only (no PANs).
- [ ] Favorites under `/me/favorites` used by Library and product detail.

## Onboarding & Name Mapping
Models: `doc/api/models/nameMappings.schema.yaml`
- [ ] `POST /name-mappings:convert` request `{ latin, locale }` and response candidates parsed.
- [ ] `POST /name-mappings/{mappingId}:select` used to persist selection.

## Catalog & Content (Public)
Models: `templates.schema.yaml`, `fonts.schema.yaml`, `materials.schema.yaml`, `products.schema.yaml`, `content.guides.schema.yaml`, `content.pages.schema.yaml`
- [ ] Templates/fonts/materials/products list/detail endpoints power Home/Shop screens.
- [ ] Guides/pages endpoints used by Guides and Legal screens with `lang` query.

## Designs (Library)
Models: `designs.schema.yaml`, `designs.version.schema.yaml`, `assets.schema.yaml`
- [ ] `POST /designs` (typed/upload/logo) creation payload defined; returns `designId`.
- [ ] `GET /designs` and `GET /designs/{designId}` used for Library list/detail.
- [ ] Versions endpoints consumed for history and diff UI.
- [ ] Duplicate via `POST /designs/{id}/duplicate`.
- [ ] Export uses `POST /assets:signed-upload` (for uploads) and `POST /assets/{id}:signed-download` for secure downloads.
- [ ] Registrability check via `POST /designs/{id}:registrability-check` mapped to UI hints.

## AI Suggestions
Models: `aiJobs.schema.yaml`, `designs.aiSuggestion.schema.yaml`
- [ ] Request `POST /designs/{id}/ai-suggestions` payload (method/model) validated.
- [ ] Poll `GET /designs/{id}/ai-suggestions` or `/{suggestionId}` with backoff until `status=completed|failed`.
- [ ] Accept/Reject actions wired to `:accept`/`:reject` endpoints.

## Cart & Checkout
Models: `carts.header.schema.yaml`, `carts.item.schema.yaml`, `promotions.schema.yaml`, `promotions.usage.schema.yaml`, `orders.payment.schema.yaml`
- [ ] Lazy cart retrieval via `GET /cart` (header) and `GET/POST /cart/items` for lines.
- [ ] `PATCH /cart` for currency/shipping address/promo hints.
- [ ] Estimate totals via `POST /cart:estimate`; totals fields mapped to UI (subtotal/discount/tax/shipping/total).
- [ ] Promotions apply/remove via `POST /cart:apply-promo` / `DELETE /cart:remove-promo`.
- [ ] PSP session via `POST /checkout/session`; confirm via `POST /checkout/confirm`.
- [ ] Idempotency-Key used for line mutations and confirm.

## Orders, Payments, Shipments, Production
Models: `orders.schema.yaml`, `orders.payment.schema.yaml`, `orders.shipment.schema.yaml`, `orders.productionEvent.schema.yaml`, `invoices.schema.yaml`
- [ ] Orders list/detail supports pagination and filtering.
- [ ] Reorder via `POST /orders/{id}:reorder` uses `designSnapshot`.
- [ ] Payments/shipment endpoints provide line items, tracking events; map to tracking UI.
- [ ] Request invoice via `POST /orders/{id}:request-invoice`; download via assets if applicable.

## Reviews
Models: `reviews.schema.yaml`
- [ ] `POST /reviews` payload validated (orderId, rating, comment).
- [ ] `GET /reviews?orderId=...` used to show user’s review state.

## Promotions (Public + User)
Models: `promotions.schema.yaml`, `promotions.usage.schema.yaml`
- [ ] Public promo visibility via `GET /promotions/{code}/public` used for hints before auth.
- [ ] Usage limits enforced server-side; UI handles errors from apply/remove.

## Assets
Models: `assets.schema.yaml`, `storage-layout.md`
- [ ] Signed upload/download flows implemented with correct `kind/purpose` and content-type.
- [ ] Expiry and size limits handled in UI with errors surfaced.

## Error Handling & Retries
- [ ] 401/403 route to login; preserve `next` deep link.
- [ ] 409 idempotency/duplicate handled with user feedback; retries disabled for non-idempotent ops.
- [ ] 429/5xx use exponential backoff with jitter; show retry affordance.

## Telemetry & Correlation
- [ ] Send `X-Client-Version` and `X-Device` headers; log request IDs returned by backend for support.

## Acceptance Criteria (MVP)
- [ ] All app flows in `doc/app/app_design.md` have corresponding endpoint coverage above.
- [ ] Field names/types match model schemas under `doc/api/models/`.
- [ ] Error cases tested per flow (auth required, invalid promo, payment fail, not found).
- [ ] Pagination and empty states verified for lists.
- [ ] Idempotency verified for mutating endpoints from the app.

References
- API Design: `doc/api/api_design.md`
- Models: `doc/api/models/*`
