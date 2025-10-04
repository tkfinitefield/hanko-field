# Develop navigation shell with bottom tabs (`作成/ショップ/注文/マイ印鑑/プロフィール`), nested navigators, and deep link support.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 015

## Goal
Implement bottom tab navigation with nested stacks and deep linking.

## Implementation Steps
1. Use `GoRouter` or manual `Navigator` with `IndexedStack` to preserve tab state.
2. Maintain navigator keys per tab; handle back button behavior (Android hardware back, iOS gesture).
3. Support deep links and push notifications routing to correct tab/stack.
4. Expose navigation helpers via Riverpod for view models.
