# Create work order view (`/admin/production/workorders/{orderId}`) summarizing design assets, instructions, and tasks.

**Parent Section:** 5. Orders & Operations > 5.3 Production & Workshop
**Task ID:** 035

## Goal
Create printable/digital work order view for production team.

## Implementation Steps
1. Show design assets, customer instructions, materials, due dates.
2. Provide buttons to mark steps complete (calls production events endpoint).
3. Optionally render PDF for printing.

## UI Components
- **Page shell:** `AdminLayout` breadcrumbs pointing back to queue and order detail.
- **Header card:** `SummaryCard` with order id, due time, responsible team, and action `ButtonGroup`.
- **Tabbed body:** `UnderlineTabs` for Overview, Assets, Instructions, Activity.
- **Assets grid:** `MediaGrid` listing design files with preview thumbnails and download `IconButton`.
- **Instruction panel:** `RichTextPanel` with step list, checkboxes, and safety callouts using `InlineAlert`.
- **Activity timeline:** `TimelineList` capturing updates, QC outcomes, and attachments.
