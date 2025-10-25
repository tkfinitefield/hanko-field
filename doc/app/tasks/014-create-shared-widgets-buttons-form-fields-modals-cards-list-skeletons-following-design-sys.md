# Create shared widgets (buttons, form fields, modals, cards, list skeletons) following design system.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 014

## Goal
Build reusable component library aligned with design system.

## Components
- Buttons (primary/secondary/ghost) with loading states and icon support.
- Form fields and validation messaging.
- Cards, list tiles, modals, bottom sheets.
- Shimmer/skeleton loaders and empty-state widgets.
- Responsive layout helpers.

## Implementation Summary
- Added a `core/ui` module that centralizes responsive helpers (`ResponsiveLayout`, `ResponsivePagePadding`) and re-exports all shared widgets via `core/ui/ui.dart` for convenient imports.
- Implemented tokens-aware primitives:
  - `AppButton` (primary/secondary/ghost variants, 3 sizes, icon slots, loading indicator, full-width support).
  - `AppTextField`, `AppValidationMessage` for labeled inputs, helper copy, and inline validation badges.
  - `AppCard` + `AppListTile` with consistent radius/elevation/border logic for elevated, outlined, and filled variants.
  - `AppModal`, `AppModalAction`, `showAppModal`, `showAppBottomSheet` utilities for dialogs and sheets.
  - Loading states via `AppShimmer`, `AppSkeletonBlock`, and `AppListSkeleton`.
  - Feedback states with `AppEmptyState` (icon/title/body + stacked primary/secondary actions).
- Updated the sample counter screen to exercise the component set (card layout, responsive padding, form field validation, primary/ghost buttons, bottom sheet trigger, list skeleton, empty state).

## Files of Note
- `app/lib/core/ui/layout/responsive_layout.dart`
- `app/lib/core/ui/widgets/*` (buttons, text fields, cards/list tiles, modals, empty state, skeletons)
- `app/lib/core/ui/ui.dart` (barrel export)
- `app/lib/features/sample_counter/presentation/counter_screen.dart` (demo integration)

## Testing
- `cd app && flutter test`
