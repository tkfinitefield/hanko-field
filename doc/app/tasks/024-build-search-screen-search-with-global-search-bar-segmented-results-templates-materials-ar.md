# Build search screen (`/search`) with global search bar, segmented results (templates/materials/articles/FAQ), and search history.

**Parent Section:** 4. Home & Discovery
**Task ID:** 024

## Goal
Implement global search across templates, materials, articles, FAQ.

## Implementation Steps
1. Provide search bar with real-time suggestions and history chips.
2. Use provider families to fetch results per category concurrently.
3. Implement segmented control to switch between result types, with infinite scroll where needed.
4. Handle voice input or barcode scanning if future requirement.

## Material Design 3 Components
- **App bar:** `Small top app bar` integrating a `Search bar` with voice `Icon button`.
- **Filters:** `Segmented buttons` to pivot between templates, materials, articles, and FAQ.
- **Results:** `List items` with leading thumbnails and supporting text for metadata.
- **Footer:** `Navigation bar` retained for primary destinations.
