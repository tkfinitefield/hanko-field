# Implement kanji dictionary (`/kanji/dictionary`) with search, favorites, and integration with design input.

**Parent Section:** 10. Guides & Cultural Content
**Task ID:** 061

## Goal
Implement kanji dictionary search screen.

## Implementation Steps
1. Provide search input with suggestions, favorites, and history.
2. Display kanji detail (meaning, stroke order, usage examples).
3. Integrate with design input to prefill selection.

## Material Design 3 Components
- **App bar:** `Small top app bar` with `Search bar` inline and bookmark `Icon button`.
- **Filters:** `Assist chips` for grade, stroke count, radical.
- **Result list:** `Supporting list items` showing glyph, readings, and actions.
- **Detail action:** `Modal bottom sheet` presenting full definition with `Filled button` to insert into editor.
