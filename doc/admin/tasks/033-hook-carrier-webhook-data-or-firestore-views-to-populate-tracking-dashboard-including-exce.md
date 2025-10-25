# Hook carrier webhook data or Firestore views to populate tracking dashboard, including exception badges and SLA indicators. ✅

**Parent Section:** 5. Orders & Operations > 5.2 Shipments & Tracking
**Task ID:** 033

## Goal
Connect the `/admin/shipments/tracking` dashboard to real carrier data streamed via webhooks and persisted in Firestore, so ops can triage SLA breaches and exceptions in near real time.

## Implementation Highlights
1. **Firestore-backed service** – Added `admin/internal/admin/shipments/FirestoreService` that queries the aggregated tracking view (default collection `ops_tracking_shipments`) and maps each document into the existing `TrackingShipment` model. Derived fields (status labels/tones, SLA badges, exception icons) are normalised when the upstream document omits them so the UI always receives complete data.
2. **Metadata-driven cache invalidation** – Each response consults a metadata document (`ADMIN_SHIPMENTS_TRACKING_METADATA_DOC`) that webhook processors update on every ingest. A lightweight in-memory cache (default TTL 15s) is reused only while the metadata `updatedAt/version` remain unchanged, so webhook arrivals invalidate the dataset without hammering Firestore.
3. **Alert surface** – Optional alert documents from `ADMIN_SHIPMENTS_TRACKING_ALERTS_COLLECTION` raise carrier outages or SLA clusters. When the collection is empty, synthetic alerts are generated from the current summary (e.g., “SLA遅延リスク”) to keep the banner area informative.
4. **Auto-refresh hints** – The dashboard conveys `TrackingSummary.LastRefresh` and `RefreshInterval` based on metadata so the front-end can display a realistic timer even if Firestore lacks explicit guidance (falls back to 30s).
5. **Configuration knobs** – New env vars allow operators to point at emulator projects, tune cache TTL / fetch limits, and define which Firestore collections back shipments vs. alerts. Defaults keep local development zero-config while production can override per environment.

## Data Flow
1. Carrier webhooks land in Cloud Run / Functions (task 098) and normalise payloads, storing denormalised tracking rows inside `ops_tracking_shipments` along with SLA enrichments and order references.
2. The same webhook job updates the metadata doc (`updatedAt`, `refreshIntervalSeconds`, optional `version` token) and pushes alert docs when widespread issues are detected.
3. Admin requests call `FirestoreService.ListTracking`, which:
   - loads metadata (cheap doc read),
   - reuses or refreshes the cached shipment slice according to the version/TTL,
   - applies query filters (status, carrier, lane, region, delay buckets) in-memory,
   - builds summaries/filters plus alert payloads, and
   - returns results to the templ/htmx handlers.

## SLA / Exception Surfacing
- SLA badges derive from explicit Firestore fields when present, otherwise fall back to delay thresholds (>=180 min ⇒ `SLA逸脱`, >=60 ⇒ `遅延リスク`, else `SLA内`).
- Exception rows automatically receive `⚠️` icons and “要対応” badges if their status is `exception` even when upstream omits a label.
- Alerts section highlights both Firestore-authored notices and synthetic summaries so the top-of-page stack always reflects current breach counts.

## Config Reference

| Variable | Purpose | Default |
| --- | --- | --- |
| `ADMIN_FIRESTORE_PROJECT_ID` | Explicit Firestore project for tracking (falls back to `FIRESTORE_PROJECT_ID` → `FIREBASE_PROJECT_ID`). | `""` |
| `ADMIN_SHIPMENTS_TRACKING_COLLECTION` | Collection containing aggregated tracking docs. | `ops_tracking_shipments` |
| `ADMIN_SHIPMENTS_TRACKING_ALERTS_COLLECTION` | Optional alert collection powering banner stack. | `ops_tracking_alerts` |
| `ADMIN_SHIPMENTS_TRACKING_METADATA_DOC` | Document path whose `updatedAt`/`version` invalidates caches. | `""` |
| `ADMIN_SHIPMENTS_TRACKING_FETCH_LIMIT` | Maximum rows pulled per refresh (capped to active shipments). | `500` |
| `ADMIN_SHIPMENTS_TRACKING_ALERTS_LIMIT` | Max alert banners shown. | `5` |
| `ADMIN_SHIPMENTS_TRACKING_CACHE_TTL` | In-memory cache TTL for shipment slice. | `15s` |
| `ADMIN_SHIPMENTS_TRACKING_REFRESH_INTERVAL` | Fallback UI refresh interval when metadata lacks value. | `30s` |

## Acceptance
- Admin dashboard now renders Firestore-fed shipments with accurate SLA/exception badges.
- Successive requests reuse cached data until metadata advertises a newer version or TTL expires, keeping Firestore costs predictable.
- Alert banners + table badges immediately reflect webhook updates thanks to metadata invalidation and derived fallbacks.
