# Create shipment batch page (`/admin/shipments/batches`) for label generation workflows and integration with shipment POST endpoints.

**Parent Section:** 5. Orders & Operations > 5.2 Shipments & Tracking
**Task ID:** 031

## Goal
Manage shipment batch creation and label generation.

## Implementation Steps
1. Page lists pending orders requiring labels with filters.
2. Provide selection and action to call `POST /admin/orders/{id}/shipments` (per order or batch aggregated endpoint).
3. Display label generation status and download links; allow retry on failure.
4. Integrate with carrier options (service level, package details).

## UI Components
- **Page shell:** `AdminLayout` + `PageHeader` including outstanding batches KPI chips.
- **Batch filters:** `FilterToolbar` (carrier `Select`, facility `Combobox`, status `ChipGroup`, date `RangePicker`).
- **Batch table:** `DataTable` with expandable rows showing manifest counts, label status, and action `ButtonGroup`.
- **Progress drawer:** `DetailDrawer` showing job timeline, assigned operator, print history.
- **Toolbar actions:** Primary `FilledButton` for Create batch, secondary `GhostButton` for bulk label regenerate.
- **Per-row metrics:** Inline `ProgressBar` and SLA `Badge` components in table cells.
