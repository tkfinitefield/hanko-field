# Establish Riverpod usage guidelines (Notifier/AsyncNotifier, providers scoping) and dependency injection strategy without code generation.

**Parent Section:** 0. Planning & Architecture
**Task ID:** 003

## Goal
Document Riverpod usage patterns without relying on code generation or `StateProvider`.

## Topics
- Provider categories (global app-level, feature-level, ephemeral UI) and lifecycle management.
- Dependency injection using provider overrides for testing.
- Handling asynchronous state with `AsyncNotifier` and custom state classes.
- Error propagation, retry patterns, and UI binding guidance.
- Best practices for provider naming, file splitting, and avoiding circular dependencies.
